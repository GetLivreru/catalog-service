package binding

import (
	"net/http"

	"github.com/go-playground/form/v4"
)

var formDecoder = form.NewDecoder()

type queryBinding struct{}

var _ Binding = queryBinding{}

func (queryBinding) Name() string { return "URL-QUERY" }
func (queryBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.URL == nil {
		return &bindingError{msg: "invalid request"}
	}
	if err := formDecoder.Decode(obj, req.URL.Query()); err != nil {
		return &bindingError{msg: err.Error()}
	}
	if err := validate(obj); err != nil {
		return &bindingError{msg: err.Error()}
	}
	return nil
}
