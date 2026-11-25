package api

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "statistical_measures/docs"
	"statistical_measures/internal/app/auth"
	"statistical_measures/internal/app/handler"
	"statistical_measures/internal/app/repository"
)

func StartServer() {
	log.Println("Starting server")
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	repo, err := repository.NewRepository()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	authService, err := auth.New(ctx)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer authService.Close()
	h := handler.NewHandler(repo, authService)
	optional := authService.Middleware(false)
	required := authService.Middleware(true)

	// Legacy SSR endpoints from labs 1–2: reads remain public;
	// draft and write endpoints also enforce JWT to prevent an authorization bypass.
	r.GET("/feed-statistical-measures", h.Feed)
	r.GET("/statistical-measures", h.Tiles)
	r.GET("/add-statistical-measures", required, h.Add)
	r.POST("/add-statistical-measures", required, h.CreateDraft)
	r.POST("/publish-statistical-measures", required, h.PublishDraft)
	r.POST("/delete-statistical-measures", required, h.DeleteMeasure)

	api := r.Group("/api")
	api.GET("/statistical-measures", optional, h.APIMeasures)
	api.GET("/statistical-measures/feed", optional, h.APIFeed)
	api.GET("/statistical-measures/draft", required, h.APIDraft)
	api.POST("/statistical-measures", required, h.APICreateMeasure)
	api.PUT("/statistical-measures/:id/publish", required, h.APIPublishMeasure)
	api.DELETE("/statistical-measures/:id", required, h.APIDeleteMeasure)
	api.POST("/statistical-measures/:id/like", required, h.APILike)
	api.POST("/users/register", h.APIRegister)
	api.POST("/users/auth", h.APIAuth)
	api.POST("/users/logout", required, h.APILogout)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	log.Println("Swagger: http://localhost:8080/swagger/index.html")
	if err := r.Run("127.0.0.1:8080"); err != nil {
		log.Fatal(err)
	}
}
