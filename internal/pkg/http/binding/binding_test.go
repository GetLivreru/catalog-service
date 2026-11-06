package binding

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/GetLivreru/catalog-service/internal/app/entity"
	"github.com/GetLivreru/catalog-service/internal/pkg/http/httph"
)

func TestJSONBinding(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		to         func() any
		invalid    bool
	}{
		{"valid category", `{"name":"Телефоны"}`, func() any { return &entity.RequestCategoryCreate{} }, false},
		{"unicode two characters", `{"name":"Яя"}`, func() any { return &entity.RequestCategoryCreate{} }, false},
		{"missing name", `{}`, func() any { return &entity.RequestCategoryCreate{} }, true},
		{"short name", `{"name":"Я"}`, func() any { return &entity.RequestCategoryCreate{} }, true},
		{"long name", `{"name":"` + strings.Repeat("a", 256) + `"}`, func() any { return &entity.RequestCategoryCreate{} }, true},
		{"malformed JSON", `not json`, func() any { return &entity.RequestCategoryCreate{} }, true},
		{"empty body", ``, func() any { return &entity.RequestCategoryCreate{} }, true},
		{"partial update", `{"name":"New"}`, func() any { return &entity.RequestProductUpdate{} }, false},
		{"zero update price", `{"price":0}`, func() any { return &entity.RequestProductUpdate{} }, false},
		{"negative update price", `{"price":-1}`, func() any { return &entity.RequestProductUpdate{} }, true},
		{"long description", `{"description":"` + strings.Repeat("a", 1001) + `"}`, func() any { return &entity.RequestProductUpdate{} }, true},
		{"missing category", `{"name":"Phone","price":100}`, func() any { return &entity.RequestProductCreate{} }, true},
		{"zero UUID", `{"name":"Phone","price":100,"category_guid":"00000000-0000-0000-0000-000000000000"}`, func() any { return &entity.RequestProductCreate{} }, true},
		{"no filters", `{}`, func() any { return &entity.RequestProductList{} }, false},
		{"valid range", `{"min_price":100,"max_price":500}`, func() any { return &entity.RequestProductList{} }, false},
		{"zero lower bound", `{"min_price":0}`, func() any { return &entity.RequestProductList{} }, true},
		{"negative upper bound", `{"max_price":-1}`, func() any { return &entity.RequestProductList{} }, true},
		{"reversed range allowed", `{"min_price":500,"max_price":100}`, func() any { return &entity.RequestProductList{} }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ScanAndValidateJSON(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body)), tc.to())
			if (err != nil) != tc.invalid {
				t.Fatalf("error=%v, invalid=%v", err, tc.invalid)
			}
			if err != nil {
				assertBadRequest(t, err)
			}
		})
	}
	for _, r := range []*http.Request{nil, {}} {
		err := ScanAndValidateJSON(r, &entity.RequestCategoryCreate{})
		if err == nil {
			t.Fatal("expected invalid request")
		}
		assertBadRequest(t, err)
	}
}

func assertBadRequest(t *testing.T, err error) {
	t.Helper()
	w := httptest.NewRecorder()
	httph.HandleError(w, httptest.NewRequest("GET", "/", nil), err)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d: %s", w.Code, w.Body.String())
	}
}

func TestQueryBinding(t *testing.T) {
	type query struct {
		Min *int64 `form:"min_price" binding:"omitempty,gt=0"`
	}
	for _, tc := range []struct {
		raw     string
		invalid bool
	}{
		{"", false}, {"?min_price=10", false}, {"?min_price=0", true}, {"?min_price=-1", true}, {"?min_price=bad", true},
	} {
		var obj query
		err := ScanAndValidateQuery(httptest.NewRequest(http.MethodGet, "/"+tc.raw, nil), &obj)
		if (err != nil) != tc.invalid {
			t.Fatalf("%s: %v", tc.raw, err)
		}
		if err != nil {
			assertBadRequest(t, err)
		}
		if tc.raw == "?min_price=10" && (obj.Min == nil || *obj.Min != 10) {
			t.Fatal("query not decoded")
		}
	}
	if err := ScanAndValidateQuery(nil, &query{}); err == nil {
		t.Fatal("expected invalid request")
	}
}

func TestValidatorReflectionAndConcurrentInit(t *testing.T) {
	v := &defaultValidator{}
	type value struct {
		Name string `binding:"required"`
	}
	valid := value{Name: "ok"}
	ptr := &valid
	for _, obj := range []any{nil, (*value)(nil), 42, "text", []value{}, valid, &valid, &ptr} {
		if err := v.ValidateStruct(obj); err != nil {
			t.Fatalf("%T: %v", obj, err)
		}
	}
	if err := v.ValidateStruct(value{}); err == nil {
		t.Fatal("missing validation")
	}
	concurrent := &defaultValidator{}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := concurrent.ValidateStruct(valid); err != nil {
				t.Error(err)
			}
			if concurrent.Engine() == nil {
				t.Error("nil engine")
			}
		}()
	}
	wg.Wait()
}

// Bind must return an HTTP-aware error even without the public wrapper.
func TestBindingErrorContract(t *testing.T) {
	type query struct {
		Min int64 `form:"min_price" binding:"gt=0"`
	}
	for _, tc := range []struct {
		name    string
		binder  Binding
		request *http.Request
		target  any
		message string
	}{
		{"nil JSON request", jsonBinding{}, nil, &entity.RequestCategoryCreate{}, "invalid request"},
		{"nil JSON body", jsonBinding{}, &http.Request{}, &entity.RequestCategoryCreate{}, "invalid request"},
		{"malformed JSON", jsonBinding{}, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json")), &entity.RequestCategoryCreate{}, "incorrect parameters"},
		{"wrong JSON type", jsonBinding{}, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":123}`)), &entity.RequestCategoryCreate{}, "incorrect parameters"},
		{"empty JSON body", jsonBinding{}, httptest.NewRequest(http.MethodPost, "/", nil), &entity.RequestCategoryCreate{}, "incorrect parameters"},
		{"validation message", jsonBinding{}, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"A"}`)), &entity.RequestCategoryCreate{}, "Key: 'RequestCategoryCreate.Name' Error:Field validation for 'Name' failed on the 'min' tag"},
		{"nil query request", queryBinding{}, nil, &query{}, "invalid request"},
		{"query decoding", queryBinding{}, httptest.NewRequest(http.MethodGet, "/?min_price=bad", nil), &query{}, ""},
		{"query validation", queryBinding{}, httptest.NewRequest(http.MethodGet, "/?min_price=0", nil), &query{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.binder.Bind(tc.request, tc.target)
			if err == nil {
				t.Fatal("expected binding error")
			}
			if tc.message != "" && err.Error() != tc.message {
				t.Fatalf("error=%q want=%q", err.Error(), tc.message)
			}
			assertBadRequest(t, err)
		})
	}
}
