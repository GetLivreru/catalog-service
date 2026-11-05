package httph_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/httph"
)

func TestErrorContext(t *testing.T) {
	original := httptest.NewRequest(http.MethodGet, "/", nil)
	httph.ErrorApply(original, entity.ErrNotFound)
	httph.ErrorApplyStatusCode(original, 404)
	if httph.ErrorGet(original) != nil || httph.ErrorGetStatusCode(original) != 0 {
		t.Fatal("unprepared request must be a no-op")
	}
	r := httph.ErrorPrepare(original)
	wrapped := fmt.Errorf("repository: %w", entity.ErrNotFound)
	w := httptest.NewRecorder()
	httph.HandleError(w, r, wrapped)
	if httph.ErrorGet(r) != wrapped || httph.ErrorGetStatusCode(r) != 404 {
		t.Fatal("original error/status missing from context")
	}
	if w.Body.String() != "{\"error\":\"not found\"}\n" {
		t.Fatal(w.Body.String())
	}
	if httph.ErrorGet(httph.ErrorPrepare(original)) != nil {
		t.Fatal("request state leaked")
	}
	if httph.ErrorGet(original) != nil {
		t.Fatal("original request modified")
	}
}

func TestErrorMiddleware(t *testing.T) {
	sentinel := errors.New("database failed")
	handler := httph.NewErrorMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httph.HandleError(w, r, sentinel)
		if httph.ErrorGet(r) != sentinel || httph.ErrorGetStatusCode(r) != 500 {
			t.Fatal("missing error context")
		}
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 500 {
		t.Fatal(w.Code)
	}
}
