package binding

import (
	"net/http"

	"github.com/GetLivreru/catalog-service/internal/pkg/http/httph"
)

type jsonBinding struct{}

var _ Binding = jsonBinding{}

func (jsonBinding) Name() string { return "JSON" }
func (jsonBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return &bindingError{msg: "invalid request"}
	}
	if err := httph.DecodeJSON(req, obj); err != nil {
		return &bindingError{msg: "incorrect parameters"}
	}
	if err := validate(obj); err != nil {
		return &bindingError{msg: err.Error()}
	}
	return nil
}
