package validation

import "github.com/labstack/echo/v4"

// BindAndValidate is the standard request binding+validation helper for handlers.
func BindAndValidate(c echo.Context, dst any) error {
	if err := c.Bind(dst); err != nil {
		return bindError(err)
	}

	if err := getValidator().Struct(dst); err != nil {
		return validationError(err)
	}

	return nil
}
