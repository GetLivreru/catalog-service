package mzerolog_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
	"github.com/GetLivreru/catalog-service/internal/app/util"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/httph"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/mzerolog"
	"github.com/rs/zerolog"
)

func TestRequestLogging(t *testing.T) {
	for _, tc := range []struct {
		name, path           string
		level                zerolog.Level
		fail, skip, expected bool
	}{
		{"debug success", "/v1/category/list", zerolog.DebugLevel, false, false, true},
		{"debug error", "/v1/category/bad", zerolog.DebugLevel, true, false, true},
		{"info success suppressed", "/v1/category/list", zerolog.InfoLevel, false, false, false},
		{"info error retained", "/v1/category/bad", zerolog.InfoLevel, true, false, true},
		{"health skipped", "/health", zerolog.DebugLevel, false, true, false},
		{"debug skipped", "/debug/pprof", zerolog.DebugLevel, false, true, false},
		{"metrics skipped", "/metrics", zerolog.DebugLevel, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := zerolog.New(&output).Level(tc.level)
			opts := []mzerolog.Option{mzerolog.WithLogger(logger), mzerolog.WithSkipper(nil)}
			if tc.skip {
				opts = append(opts, mzerolog.WithSkipper(util.IsFilteredHttpRoute))
			}
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if tc.fail {
					httph.HandleError(w, r, fmt.Errorf("service: %w", entity.ErrIncorrectParameters))
					return
				}
				w.WriteHeader(http.StatusOK)
			})
			handler := httph.NewErrorMiddleware()(mzerolog.NewMiddleware(opts...)(next))
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			req.RemoteAddr = "127.0.0.1:1234"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if !called {
				t.Fatal("skipper suppressed handler")
			}
			if !tc.expected {
				if output.Len() != 0 {
					t.Fatal(output.String())
				}
				return
			}
			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			wantLevel, tail := "debug", " finished with no error"
			if tc.fail {
				wantLevel, tail = "error", " finished (or aborted) with error"
				if record["error"] != "service: incorrect parameters" {
					t.Fatal(record)
				}
			}
			if record["level"] != wantLevel || record[zerolog.MessageFieldName] != "POST "+tc.path+tail {
				t.Fatal(record)
			}
			if record["client_ip"] != req.RemoteAddr || record["exec_time"] == "" || record["exec_time"] == nil {
				t.Fatal(record)
			}
			if tc.fail && strings.Contains(w.Body.String(), "service:") {
				t.Fatal("wrapped error leaked to client")
			}
		})
	}
}
