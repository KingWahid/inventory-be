package validation

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validateOnce sync.Once
	validateInst *validator.Validate
)

func getValidator() *validator.Validate {
	validateOnce.Do(func() {
		v := validator.New()
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			tag := fld.Tag.Get("json")
			if tag == "" {
				return fld.Name
			}
			name := strings.Split(tag, ",")[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})
		validateInst = v
	})
	return validateInst
}
