package validation

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
)

func bindError(err error) error {
	details := map[string]any{
		"errors": []map[string]any{
			{
				"field":  "",
				"rule":   errorcodes.ValidationRuleBind,
				"value":  "",
				"reason": err.Error(),
			},
		},
	}
	return errorcodes.ErrValidationError.WithDetails(details)
}

func validationError(err error) error {
	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		details := map[string]any{
			"errors": []map[string]any{
				{
					"field":  "",
					"rule":   errorcodes.ValidationRuleValidate,
					"value":  "",
					"reason": err.Error(),
				},
			},
		}
		return errorcodes.ErrValidationError.WithDetails(details)
	}

	items := make([]map[string]any, 0, len(verrs))
	for _, fe := range verrs {
		items = append(items, map[string]any{
			"field":  fe.Field(),
			"rule":   fe.Tag(),
			"value":  fmt.Sprintf("%v", fe.Value()),
			"reason": fe.Error(),
		})
	}
	return errorcodes.ErrValidationError.WithDetails(map[string]any{"errors": items})
}
