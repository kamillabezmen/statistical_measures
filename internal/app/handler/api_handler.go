package handler

import (
	"net/http"
	"strconv"
	"strings"

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

	measures, err := h.repository.GetAPIPublished(calculationTime)
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
func (h *Handler) APIDraft(c *gin.Context) {
	measure, err := h.repository.GetAPIDraft()

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "черновик текущего пользователя не найден",
		})
		return
	}

	c.JSON(http.StatusOK, makeAPIResponse(*measure))
}

// 3. GET ленты.
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

	measure, err := h.repository.GetAPIFeed(id, next)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "расчёт в ленте не найден",
		})
		return
	}

	c.JSON(http.StatusOK, makeAPIResponse(*measure))
}

// 4. POST создания черновика с изображением и видео.
func (h *Handler) APICreateMeasure(c *gin.Context) {
	title := strings.TrimSpace(c.PostForm("title"))

	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "необходимо указать title",
		})
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
		videoName,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        measure.ID,
		"title":     measure.Title,
		"status":    measure.Status,
		"image_url": measure.ImageURL,
		"video_url": measure.VideoURL,
	})
}

// 5. PUT публикации черновика.
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

	err = h.repository.PublishAPIDraft(
		id,
		request.Description,
		request.Formula,
		request.CalculationTime,
	)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "расчёт опубликован",
	})
}

// 6. DELETE — логическое удаление.
func (h *Handler) APIDeleteMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "некорректный id",
		})
		return
	}

	if err := h.repository.DeleteAPIMeasure(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "можно удалить только свою опубликованную запись",
		})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "расчёт удалён",
	})
}

// 7. POST установки или снятия лайка.
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

	if request.Value != 0 && request.Value != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "value должен быть равен 0 или 1",
		})
		return
	}

	if err := h.repository.SetAPILike(id, request.Value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	message := "лайк установлен"

	if request.Value == 0 {
		message = "лайк удалён"
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: message,
	})
}

// 8. POST регистрации пользователя.
func (h *Handler) APIRegister(c *gin.Context) {
	var request dto.RegisterUserRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "необходимо передать name, email и password",
		})
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

	c.JSON(http.StatusCreated, gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	})
}

// 9. POST аутентификации — заглушка для ЛР-4.
func (h *Handler) APIAuth(c *gin.Context) {
	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "аутентификация будет реализована в лабораторной работе № 4",
	})
}

// 10. POST деавторизации — заглушка для ЛР-4.
func (h *Handler) APILogout(c *gin.Context) {
	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "деавторизация будет реализована в лабораторной работе № 4",
	})
}
