package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	"github.com/GitLivreru/catalog-service/internal/app/config/section"
)

type Config struct {
	Repository section.Repository
	Processor  section.Processor
	Monitor    section.Monitor
}

var Root Config

func Load() {
	// Загружаем переменные из .env.
	// Если файла нет — игнорируем ошибку.
	_ = godotenv.Load()

	// Заполняем Root из переменных окружения с префиксом APP.
	if err := envconfig.Process("APP", &Root); err != nil {
		log.Fatal(err)
	}
}
