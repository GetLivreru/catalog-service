package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	"github.com/GetLivreru/catalog-service/internal/app/config/section"
)

type Config struct {
	Repository section.Repository
	Processor  section.Processor
	Monitor    section.Monitor
}

var Root Config

func Load() {
	// Загружаем переменные из .env.
	// Отсутствие файла не мешает запуску с переменными окружения.
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	// Заполняем Root из переменных окружения с префиксом APP.
	if err := envconfig.Process("APP", &Root); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
}
