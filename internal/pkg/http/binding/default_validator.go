package binding

import (
	"reflect"
	"sync"

	"github.com/go-playground/validator/v10"
)

type defaultValidator struct {
	validate *validator.Validate
	once     sync.Once
}

var _ StructValidator = &defaultValidator{}

func (v *defaultValidator) lazyInit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")
	})
}

func (v *defaultValidator) Engine() any {
	v.lazyInit()
	return v.validate
}

func (v *defaultValidator) ValidateStruct(obj any) error {
	value := reflect.ValueOf(obj)
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	v.lazyInit()
	return v.validate.Struct(value.Interface())
}
