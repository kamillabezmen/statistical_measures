package handler

import (
	"errors"
	"net/http"
	"statistical_measures/internal/app/auth"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"statistical_measures/internal/app/dto"
	"statistical_measures/internal/app/repository"
	"statistical_measures/internal/app/storage"
)

func makeAPIResponse(
	measure repository.APIStatisticalMeasure,
) dto.StatisticalMeasureResponse {
	return dto.StatisticalMeasureResponse{
		ID:              measure.ID,
		Title:           measure.Title,
		Description:     measure.Description,
		Formula:         measure.Formula,
		CalculationTime: measure.CalculationTime,
		ImageURL:        measure.ImageURL,
		VideoURL:        measure.VideoURL,
		IsCreator:       measure.IsCreator,
		IsLiked:         measure.IsLiked,
		LikesCount:      measure.LikesCount,
	}
}

// 1. GET списка опубликованных расчётов с фильтрацией.
// @Summary Список опубликованных мер
// @Tags statistical-measures
// @Produce json
// @Param calculation_time query int false "Время расчёта"
// @Description Доступен гостю и авторизованному пользователю. Для гостя is_creator и is_liked равны 0.
// @Success 200 {array} dto.StatisticalMeasureResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /statistical-measures [get]
func (h *Handler) APIMeasures(c *gin.Context) {
	var calculationTime *int

	filter := c.Query("calculation_time")

	if filter != "" {
		value, err := strconv.Atoi(filter)
		if err != nil || value < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "calculation_time должен быть целым неотрицательным числом",
			})
			return
		}

		calculationTime = &value
	}

	measures, err := h.repository.GetAPIPublished(calculationTime, auth.CurrentUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "не удалось получить список расчётов",
		})
		return
	}

	response := make([]dto.StatisticalMeasureResponse, 0, len(measures))

	for _, measure := range measures {
		response = append(response, makeAPIResponse(measure))
	}

	c.JSON(http.StatusOK, response)
}

// 2. GET черновика текущего пользователя без ID.
// @Summary Черновик текущего пользователя
// @Tags statistical-measures
// @Produce json
// @Security BearerAuth
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Success 200 {object} dto.StatisticalMeasureResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /statistical-measures/draft [get]
func (h *Handler) APIDraft(c *gin.Context) {
	measure, err := h.repository.GetAPIDraft(auth.CurrentUserID(c))

	if err != nil {
		writeRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, makeAPIResponse(*measure))
}

// 3. GET ленты.
// @Summary Лента опубликованных мер
// @Tags statistical-measures
// @Produce json
// @Param id query int false "ID меры"
// @Param next query bool false "Следующая запись"
// @Description Доступен гостю и авторизованному пользователю. Для гостя is_creator и is_liked равны 0.
// @Success 200 {object} dto.StatisticalMeasureResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /statistical-measures/feed [get]
func (h *Handler) APIFeed(c *gin.Context) {
	var id *int

	idString := c.Query("id")

	if idString != "" {
		value, err := strconv.Atoi(idString)
		if err != nil || value <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "id должен быть положительным целым числом",
			})
			return
		}

		id = &value
	}

	next := strings.EqualFold(c.Query("next"), "true")

	measure, err := h.repository.GetAPIFeed(id, next, auth.CurrentUserID(c))

	if err != nil {
		writeRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, makeAPIResponse(*measure))
}

// 4. POST создания черновика с изображением и видео.
// @Summary Создание черновика
// @Tags statistical-measures
// @Produce json
// @Accept mpfd
// @Param title formData string true "Название"
// @Param image formData file true "Изображение"
// @Param video formData file true "Видео"
// @Security BearerAuth
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Success 201 {object} dto.DraftResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /statistical-measures [post]
func (h *Handler) APICreateMeasure(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<20)
	title := strings.TrimSpace(c.PostForm("title"))

	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "необходимо указать title",
		})
		return
	}

	if _, err := h.repository.GetAPIDraft(auth.CurrentUserID(c)); err == nil {
		writeRepositoryError(c, repository.ErrDraftExists)
		return
	} else if !errors.Is(err, repository.ErrNotFound) {
		writeRepositoryError(c, err)
		return
	}
	image, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "необходимо передать файл image",
		})
		return
	}

	video, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "необходимо передать файл video",
		})
		return
	}

	imageName, err := storage.UploadFile(
		c.Request.Context(),
		image,
		"image",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "не удалось загрузить изображение в MinIO",
		})
		return
	}

	videoName, err := storage.UploadFile(
		c.Request.Context(),
		video,
		"video",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "не удалось загрузить видео в MinIO",
		})
		return
	}

	measure, err := h.repository.CreateAPIDraft(
		title,
		imageName,
		videoName, auth.CurrentUserID(c),
	)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.DraftResponse{ID: measure.ID, Title: measure.Title, Status: measure.Status, ImageURL: measure.ImageURL, VideoURL: measure.VideoURL})
}

// 5. PUT публикации черновика.
// @Summary Публикация своего черновика
// @Tags statistical-measures
// @Produce json
// @Accept json
// @Param id path int true "ID меры"
// @Param request body dto.PublishMeasureRequest true "Поля публикации"
// @Security BearerAuth
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /statistical-measures/{id}/publish [put]
func (h *Handler) APIPublishMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "некорректный id",
		})
		return
	}

	var request dto.PublishMeasureRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "некорректный JSON",
		})
		return
	}

	if request.CalculationTime != nil && *request.CalculationTime < 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "calculation_time должен быть неотрицательным"})
		return
	}
	err = h.repository.PublishAPIDraft(
		id,
		request.Description,
		request.Formula,
		request.CalculationTime, auth.CurrentUserID(c),
	)

	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "расчёт опубликован",
	})
}

// 6. DELETE — логическое удаление.
// @Summary Логическое удаление своей меры
// @Tags statistical-measures
// @Produce json
// @Param id path int true "ID меры"
// @Security BearerAuth
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /statistical-measures/{id} [delete]
func (h *Handler) APIDeleteMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "некорректный id",
		})
		return
	}

	if err := h.repository.DeleteAPIMeasure(id, auth.CurrentUserID(c)); err != nil {
		writeRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "расчёт удалён",
	})
}

// 7. POST установки или снятия лайка.
// @Summary Поставить или снять лайк
// @Tags statistical-measures
// @Produce json
// @Accept json
// @Param id path int true "ID меры"
// @Param request body dto.LikeRequest true "1 — поставить, 0 — снять"
// @Security BearerAuth
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /statistical-measures/{id}/like [post]
func (h *Handler) APILike(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "некорректный id",
		})
		return
	}

	var request dto.LikeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "необходимо передать value: 0 или 1",
		})
		return
	}

	if request.Value == nil || (*request.Value != 0 && *request.Value != 1) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "value должен быть равен 0 или 1",
		})
		return
	}

	if err := h.repository.SetAPILike(id, *request.Value, auth.CurrentUserID(c)); err != nil {
		writeRepositoryError(c, err)
		return
	}

	message := "лайк установлен"

	if *request.Value == 0 {
		message = "лайк удалён"
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: message,
	})
}

// 8. POST регистрации пользователя.
// @Summary Регистрация пользователя
// @Tags users
// @Produce json
// @Accept json
// @Param request body dto.RegisterUserRequest true "Регистрация"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/register [post]
func (h *Handler) APIRegister(c *gin.Context) {
	var request dto.RegisterUserRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "необходимо передать name, email и password",
		})
		return
	}

	if strings.TrimSpace(request.Name) == "" || len(request.Password) > 72 || len(request.Name) > 100 || len(request.Email) > 100 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "проверьте имя, email и длину пароля (8–72 байта)"})
		return
	}
	user, err := h.repository.RegisterAPIUser(
		request.Name,
		request.Email,
		request.Password,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "не удалось зарегистрировать пользователя; возможно, email уже существует",
		})
		return
	}

	c.JSON(http.StatusCreated, dto.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email})
}

// 9. POST authentication.
// @Summary Аутентификация и получение JWT
// @Tags users
// @Produce json
// @Accept json
// @Param request body dto.AuthRequest true "Email и пароль"
// @Failure 401 {object} dto.ErrorResponse
// @Success 200 {object} dto.AuthResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/auth [post]
func (h *Handler) APIAuth(c *gin.Context) {
	var request dto.AuthRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "необходимо передать email и password"})
		return
	}
	user, err := h.repository.Authenticate(request.Email, request.Password)
	if errors.Is(err, repository.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "неверный email или пароль"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "ошибка аутентификации"})
		return
	}
	token, err := h.auth.Issue(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "не удалось создать токен"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, dto.AuthResponse{AccessToken: token, TokenType: "Bearer", ExpiresIn: int(auth.TokenLifetime / time.Second)})
}

// 10. POST logout: revoke the current JWT in Redis.
// @Summary Выход и отзыв JWT
// @Tags users
// @Produce json
// @Security BearerAuth
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /users/logout [post]
func (h *Handler) APILogout(c *gin.Context) {
	claims := auth.CurrentClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "требуется JWT"})
		return
	}
	if err := h.auth.Revoke(c.Request.Context(), claims); err != nil {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "не удалось отозвать токен"})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "выход выполнен"})
}

func writeRepositoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrForbidden):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Error: err.Error()})
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: err.Error()})
	case errors.Is(err, repository.ErrDraftExists), errors.Is(err, repository.ErrInvalidLike):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "ошибка базы данных"})
	}
}
