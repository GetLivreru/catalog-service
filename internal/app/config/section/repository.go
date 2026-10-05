package section

import "time"

type (
	Repository struct {
		// TODO: Какое поле здесь нужно?
		Postgres RepositoryPostgres
	}

	RepositoryPostgres struct {
		// TODO: Реализуйте поля для подключения к PostgreSQL
		// Address, Username, Password, Name — string, тег required:"true"
		// ReadTimeout, WriteTimeout — time.Duration, теги:
		//   split_words:"true" default:"30s"
		// Важно: дефолт таймаутов именно "30s" (не "5s") — как в .env / .env.example ниже
		Address      string        `required:"true"`
		Username     string        `required:"true"`
		Password     string        `required:"true"`
		Name         string        `required:"true"`
		ReadTimeout  time.Duration `split_words:"true" default:"30s"`
		WriteTimeout time.Duration `split_words:"true" default:"30s"`
	}
)
