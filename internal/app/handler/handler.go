package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"statistical_measures/internal/app/repository"
)

//Handler объединяет обработчики страниц и доступ к репозиторию.
type Handler struct {
	repository *repository.Repository
}

//MeasureView содержит данные расчёта для HTML-шаблонов.
type MeasureView struct {
	ID              int
	Title           string
	Description     string
	FullDescription string
	Formula         string
	CalculationTime int
	ImageKey        string
	VideoKey        string
	LikesCount      int
	ImageURL        string
	VideoURL        string
}

//создаёт обработчик с подключённым репозиторием.
func NewHandler(repository *repository.Repository) *Handler {
	return &Handler{
		repository: repository,
	}
}

//подготавливает данные расчёта для отображения.
func makeMeasureView(measure *repository.StatisticalMeasure) MeasureView {
	return MeasureView{
		ID:              measure.ID,
		Title:           measure.Title,
		Description:     measure.Description,
		FullDescription: measure.FullDescription,
		Formula:         measure.Formula,
		CalculationTime: measure.CalculationTime,
		ImageKey:        measure.ImageKey,
		VideoKey:        measure.VideoKey,
		LikesCount:      len(measure.Likes),
		ImageURL:        "http://localhost:9000/statistical-measures/" + measure.ImageKey,
		VideoURL:        "http://localhost:9000/statistical-measures/" + measure.VideoKey,
	}
}

//отображает выбранный или следующий опубликованный расчёт.
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

	measureView := makeMeasureView(measure)

	ctx.HTML(http.StatusOK, "feed_statistical_measures.html", gin.H{
		"measure": measureView,
	})
}

//отображает страницу с данными черновика.
func (h *Handler) Add(ctx *gin.Context) {
    measure := h.repository.GetDraft()

    if measure == nil {
        ctx.String(http.StatusNotFound, "Черновик не найден")
        return
    }

    measureView := makeMeasureView(measure)

    ctx.HTML(http.StatusOK, "add_statistical_measures.html", gin.H{
        "measure": measureView,
    })
}

//отображает опубликованные расчёты с фильтрацией по времени.
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
			ctx.String(http.StatusBadRequest, "Время вычисления не может быть отрицательным")
			return
		}

		measures = h.repository.GetPublishedByCalculationTime(maxTime)
	}
	measureViews := makeMeasureViews(measures)

	ctx.HTML(http.StatusOK, "tiles_statistical_measures.html", gin.H{
		"measures": measureViews,
		"time":     timeString,
	})
}

//подготавливает список расчётов для отображения.
func makeMeasureViews(measures []repository.StatisticalMeasure) []MeasureView {
	var result []MeasureView

	for _, measure := range measures {
		measureView := makeMeasureView(&measure)
		result = append(result, measureView)
	}

	return result
}
