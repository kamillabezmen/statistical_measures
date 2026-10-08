package api

import (
	"log"

	"github.com/gin-gonic/gin"

	"statistical_measures/internal/app/handler"
	"statistical_measures/internal/app/repository"
)

func StartServer() {
	log.Println("Starting server")

	r := gin.Default()

	// HTML-шаблоны.
	r.LoadHTMLGlob("templates/*")

	// Статические файлы SSR-сервера:
	r.Static("/static", "./resources")

	// Repository с подключением к PostgreSQL.
	repo, err := repository.NewRepository()
	if err != nil {
		log.Fatal(err)
	}

	h := handler.NewHandler(repo)

	// 3 GET запроса

	// Лента.
	r.GET("/feed-statistical-measures", h.Feed)

	// Добавление / просмотр draft.
	r.GET("/add-statistical-measures", h.Add)

	// Плитка + поиск.
	r.GET("/statistical-measures", h.Tiles)

	// 3 POST

	// Создание draft через ORM.
	r.POST("/add-statistical-measures", h.CreateDraft)

	// Публикация draft через ORM.
	r.POST("/publish-statistical-measures", h.PublishDraft)

	// Логическое удаление обычным SQL UPDATE.
	r.POST("/delete-statistical-measures", h.DeleteMeasure)

	log.Println("Server running on http://localhost:8080")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}