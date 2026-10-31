package binding

import "net/http"

func ScanAndValidateJSON(r *http.Request, to any) error  { return scanAndValidate(r, to, bJSON) }
func ScanAndValidateQuery(r *http.Request, to any) error { return scanAndValidate(r, to, bQuery) }
func scanAndValidate(r *http.Request, to any, b Binding) error {
	if err := b.Bind(r, to); err != nil {
		return &bindingError{msg: err.Error()}
	}
	return nil
}
