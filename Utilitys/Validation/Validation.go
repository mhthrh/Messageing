package Validation

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/pborman/uuid"
	"regexp"
)

type (
	Field struct {
		StructField string
		ActualTag   string
		Kind        interface{}
		Value       interface{}
		Param       interface{}
	}

	ValidationError struct {
		validator.FieldError
	}
	Validation struct {
		validate *validator.Validate
	}
	ValidationErrors []ValidationError
)

func (v ValidationError) Error() string {
	return fmt.Sprintf(
		"Key: '%s' Message: Field validation for '%s' failed on the '%s' tag",
		v.Namespace(),
		v.Field(),
		v.Tag(),
	)
}

func (v ValidationErrors) Errors() []string {
	var errs []string
	for _, err := range v {
		errs = append(errs, err.Error())
	}

	return errs
}

func NewValidation() *Validation {
	validate := validator.New()
	validate.RegisterValidation("sku", validateSKU)
	validate.RegisterValidation("uuid", ValidUUID)
	return &Validation{validate}
}

func (v *Validation) Validate(i interface{}) []Field {
	var Fields []Field
	err := v.validate.Struct(i)
	if err == nil {
		return nil
	}

	for _, err := range err.(validator.ValidationErrors) {
		Fields = append(Fields, Field{
			StructField: err.StructField(),
			ActualTag:   err.ActualTag(),
			Kind:        err.Kind(),
			Value:       err.Value(),
			Param:       err.Param(),
		})

	}
	return Fields
}
func (v *Validation) SomeValidate(i interface{}, fieldName ...string) []Field {
	var Fields []Field
	for _, s := range fieldName {
		err := v.validate.StructPartial(i, s)
		if err != nil {
			t := err.(validator.ValidationErrors)
			Fields = append(Fields, Field{
				StructField: t[0].StructField(),
				ActualTag:   t[0].ActualTag(),
				Kind:        t[0].Kind(),
				Value:       t[0].Value(),
				Param:       t[0].Param(),
			})
		}
	}

	return Fields
}

func validateSKU(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`[a-z]+-[a-z]+-[a-z]+`)
	sku := re.FindAllString(fl.Field().String(), -1)

	if len(sku) == 1 {
		return true
	}
	return false
}
func ValidUUID(fl validator.FieldLevel) bool {
	uuid := uuid.Parse(fl.Field().String())
	if uuid == nil {
		return false
	}
	return true
}
