package main

import "statistical_measures/internal/api"

// @title Statistical Measures API
// @version 4.0
// @description Веб-сервис статистических мер. Для защищённых методов нажмите Authorize и введите Bearer, пробел и access_token.
// @host localhost:8080
// @BasePath /api
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	api.StartServer()
}
