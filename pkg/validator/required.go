package validator

import "strings"

func Required(
	v *Validator,
	field string,
	value string,
) {

	if strings.TrimSpace(value) == "" {
		v.AddError(
			field,
			field+" is required",
		)
	}
}
