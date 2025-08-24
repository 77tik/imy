package utils

import (
	"errors"
	"net/http"
	"strings"

	xerrors "github.com/zeromicro/x/errors"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func (v Validator) Validate(_ *http.Request, data any) error {
	err := v.validate.Struct(data)
	if err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) { // 应该不可能
			return xerrors.New(http.StatusBadRequest, err.Error())
		}

		var errs []string
		for _, fe := range err.(validator.ValidationErrors) {
			errs = append(errs, fe.Error())
		}
		return xerrors.New(http.StatusBadRequest, strings.Join(errs, ", "))
	}

	return nil
}

func NewValidator() *Validator {
	return &Validator{
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}
