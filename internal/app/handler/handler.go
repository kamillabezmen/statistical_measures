package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"statistical_measures/internal/app/auth"
	"statistical_measures/internal/app/repository"
)

// объединяем контроллеры и доступ к репозиторию.
type Handler struct {
	repository *repository.Repository
	auth       *auth.Service
}

// MeasureView содержит данные для HTML-шаблонов.
type MeasureView struct {
	ID              int
	Title           string
	Description     string
	Formula         string
	CalculationTime int
	LikesCount      int
	ImageURL        string
	VideoURL        string
}

// NewHandler создаёт Handler.
func NewHandler(repository *repository.Repository, authService *auth.Service) *Handler {
	return &Handler{
		repository: repository,
		auth:       authService,
	}
}

// подготавливаем данные для HTML.
func makeMeasureView(measure *repository.StatisticalMeasure) MeasureView {
	view := MeasureView{
		ID:         measure.ID,
		Title:      measure.Title,
		LikesCount: measure.LikesCount,
		ImageURL:   measure.ImageURL,
		VideoURL:   measure.VideoURL,
	}

	if measure.Description != nil {
		view.Description = *measure.Description
	}

	if measure.Formula != nil {
		view.Formula = *measure.Formula
	}

	if measure.CalculationTime != nil {
		view.CalculationTime = *measure.CalculationTime
	}

	return view
}

// Получаем один опубликованный расчёт.
func (h *Handler) Feed(ctx *gin.Context) {
	idString := ctx.Query("id")

	id, err := strconv.Atoi(idString)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Некорректный ID")
		return
	}

	var measure *repository.StatisticalMeasure

	if ctx.Query("next") == "true" {
		measure = h.repository.GetNext(id)
	} else {
		measure = h.repository.GetByID(id)
	}

	if measure == nil {
		ctx.String(http.StatusNotFound, "Расчёт не найден")
		return
	}

	ctx.HTML(http.StatusOK, "feed_statistical_measures.html", gin.H{
		"measure": makeMeasureView(measure),
	})
}

// Если draft есть, показывает его, если draft нет — показывает пустую форму.
func (h *Handler) Add(ctx *gin.Context) {
	measure := h.repository.GetDraft(auth.CurrentUserID(ctx))

	if measure == nil {
		ctx.HTML(http.StatusOK, "add_statistical_measures.html", gin.H{
			"hasDraft": false,
		})
		return
	}

	ctx.HTML(http.StatusOK, "add_statistical_measures.html", gin.H{
		"hasDraft": true,
		"measure":  makeMeasureView(measure),
	})
}

// Получаем опубликованные расчёты и выполняем поиск через БД.
func (h *Handler) Tiles(ctx *gin.Context) {
	timeString := ctx.Query("time")

	var measures []repository.StatisticalMeasure

	if timeString == "" {
		measures = h.repository.GetPublished()
	} else {
		maxTime, err := strconv.Atoi(timeString)
		if err != nil {
			ctx.String(http.StatusBadRequest, "Некорректное время вычисления")
			return
		}

		if maxTime < 0 {
			ctx.String(
				http.StatusBadRequest,
				"Время вычисления не может быть отрицательным",
			)
			return
		}

		measures = h.repository.GetPublishedByCalculationTime(maxTime)
	}

	ctx.HTML(http.StatusOK, "tiles_statistical_measures.html", gin.H{
		"measures": makeMeasureViews(measures),
		"time":     timeString,
	})
}

// Создаём draft после нажатия "Далее".
func (h *Handler) CreateDraft(ctx *gin.Context) {
	title := ctx.PostForm("title")

	if title == "" {
		ctx.String(http.StatusBadRequest, "Введите название расчёта")
		return
	}

	err := h.repository.CreateDraft(title, auth.CurrentUserID(ctx))
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/add-statistical-measures")
}

// PublishDraft — POST.
// Заполняем описание, тематические поля и публикуем draft.
func (h *Handler) PublishDraft(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.PostForm("id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "Некорректный ID")
		return
	}

	description := ctx.PostForm("description")
	formula := ctx.PostForm("formula")

	calculationTime, err := strconv.Atoi(ctx.PostForm("calculation_time"))
	if err != nil || calculationTime < 0 {
		ctx.String(http.StatusBadRequest, "Некорректное время расчёта")
		return
	}

	if description == "" || formula == "" {
		ctx.String(
			http.StatusBadRequest,
			"Заполните описание и формулу",
		)
		return
	}

	err = h.repository.PublishDraft(
		id,
		description,
		formula,
		calculationTime, auth.CurrentUserID(ctx),
	)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/statistical-measures")
}

// DeleteMeasure — POST.
// Выполняем логическое удаление через SQL UPDATE.
func (h *Handler) DeleteMeasure(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.PostForm("id"))
	if err != nil {
		ctx.String(http.StatusBadRequest, "Некорректный ID")
		return
	}

	err = h.repository.DeleteMeasure(id, auth.CurrentUserID(ctx))
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/statistical-measures")
}

// makeMeasureViews подготавливает список для HTML.
func makeMeasureViews(measures []repository.StatisticalMeasure) []MeasureView {
	var result []MeasureView

	for _, measure := range measures {
		result = append(result, makeMeasureView(&measure))
	}

	return result
}
