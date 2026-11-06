package config

import (
	"io"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/GetLivreru/catalog-service/internal/app/config/section"
)

type Config struct {
	Repository section.Repository
	Processor  section.Processor
	Monitor    section.Monitor
}

var Root Config

type LoadArgs struct {
	Output          io.Writer `json:"-"`
	EnableSimpleLog bool
}

func createLogger(level zerolog.Level, output io.Writer) zerolog.Logger {
	return zerolog.New(output).Level(level).With().Timestamp().Logger()
}

func Load(args LoadArgs) {
	zerolog.TimestampFieldName = "timestamp"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = time.RFC3339
	if args.Output == nil {
		args.Output = os.Stdout
	}
	if args.EnableSimpleLog {
		args.Output = zerolog.ConsoleWriter{Out: args.Output}
	}
	log.Logger = createLogger(zerolog.DebugLevel, args.Output)
	log.Debug().Msg("Logger initialized with Debug level")
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("No .env file found")
	}
	if err := envconfig.Process("APP", &Root); err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}
	level, err := zerolog.ParseLevel(Root.Monitor.LogLevel)
	if err != nil {
		log.Warn().Err(err).Str("log_level", Root.Monitor.LogLevel).Msg("Invalid log level, falling back to debug")
		level = zerolog.DebugLevel
	}
	log.Logger = createLogger(level, args.Output)
	log.Info().Str("log_level", level.String()).Msg("Logger re-initialized with config level")
}
