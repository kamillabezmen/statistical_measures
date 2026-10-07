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

	// Подключаем HTML-шаблоны
	r.LoadHTMLGlob("templates/*")

	r.Static("/static", "./resources")

	// Создаём Repository
	repo, err := repository.NewRepository()
	if err != nil {
		log.Fatal(err)
	}

	// Создаём Handler и передаём ему Repository
	h := handler.NewHandler(repo)

	// Страница ленты
	r.GET("/feed-statistical-measures", h.Feed)

	// Страница добавления
	r.GET("/add-statistical-measures", h.Add)
    // Страница плитки
	r.GET("/statistical-measures", h.Tiles)

	// Запускаем сервер на localhost:8080
	r.Run()

	log.Println("Server down")
}