package binding

import (
	"errors"
	"net/http"

	"github.com/go-playground/form/v4"
)

var formDecoder = form.NewDecoder()

type queryBinding struct{}

func (queryBinding) Name() string { return "URL-QUERY" }
func (queryBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.URL == nil {
		return errors.New("invalid request")
	}
	if err := formDecoder.Decode(obj, req.URL.Query()); err != nil {
		return err
	}
	return validate(obj)
}
