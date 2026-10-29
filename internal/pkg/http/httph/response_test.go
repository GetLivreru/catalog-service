package httph_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/httph"
)

func TestHandleError(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"wrapped domain error", fmt.Errorf("transaction: %w", entity.ErrNotFound), 404, "not found"},
		{"conflict", entity.ErrAlreadyExists, 409, "already exists"},
		{"invalid parameters", entity.ErrIncorrectParameters, 400, "incorrect parameters"},
		{"unknown error", errors.New("unexpected"), 500, "unexpected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			httph.HandleError(w, tc.err)
			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != tc.status || body["error"] != tc.message {
				t.Fatalf("status=%d body=%v", w.Code, body)
			}
			if w.Header().Get("Content-Type") != "application/json" {
				t.Fatal("missing JSON content type")
			}
		})
	}
}
