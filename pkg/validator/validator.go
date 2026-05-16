package validator

type Validator struct {
	Errors ValidationErrors
}

func New() *Validator {
	return &Validator{
		Errors: ValidationErrors{},
	}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(
	field string,
	message string,
) {

	_, exists := v.Errors[field]

	if exists {
		return
	}

	v.Errors[field] = message
}
