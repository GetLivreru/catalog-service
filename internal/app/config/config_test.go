package config

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestLoadLoggerStages(t *testing.T) {
	oldRoot, oldLogger := Root, log.Logger
	oldTimestamp, oldMessage, oldFormat := zerolog.TimestampFieldName, zerolog.MessageFieldName, zerolog.TimeFieldFormat
	t.Cleanup(func() {
		Root = oldRoot
		log.Logger = oldLogger
		zerolog.TimestampFieldName = oldTimestamp
		zerolog.MessageFieldName = oldMessage
		zerolog.TimeFieldFormat = oldFormat
	})
	t.Setenv("APP_REPOSITORY_POSTGRES_ADDRESS", "localhost:5432")
	t.Setenv("APP_REPOSITORY_POSTGRES_USERNAME", "test")
	t.Setenv("APP_REPOSITORY_POSTGRES_PASSWORD", "test")
	t.Setenv("APP_REPOSITORY_POSTGRES_NAME", "test")
	for _, level := range []string{"debug", "info"} {
		t.Run(level, func(t *testing.T) {
			t.Setenv("APP_MONITOR_LOG_LEVEL", level)
			var output bytes.Buffer
			Load(LoadArgs{Output: &output})
			log.Debug().Msg("after-config-debug")
			log.Error().Msg("after-config-error")
			records := strings.Split(strings.TrimSpace(output.String()), "\n")
			if len(records) < 3 {
				t.Fatal(output.String())
			}
			foundDebug, foundError := false, false
			for i, line := range records {
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record["timestamp"] == nil || record["msg"] == nil || record["time"] != nil || record["message"] != nil {
					t.Fatal(record)
				}
				if i == 0 && (record["level"] != "debug" || record["msg"] != "Logger initialized with Debug level") {
					t.Fatal(record)
				}
				foundDebug = foundDebug || record["msg"] == "after-config-debug"
				foundError = foundError || record["msg"] == "after-config-error"
			}
			if foundDebug != (level == "debug") || !foundError {
				t.Fatal(output.String())
			}
		})
	}
	t.Run("console", func(t *testing.T) {
		var output bytes.Buffer
		Load(LoadArgs{Output: &output, EnableSimpleLog: true})
		if !strings.Contains(output.String(), "Logger initialized with Debug level") {
			t.Fatal(output.String())
		}
	})
}
