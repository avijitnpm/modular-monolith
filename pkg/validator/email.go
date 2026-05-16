package validator

import "strings"

func Email(
	v *Validator,
	field string,
	value string,
) {

	if !strings.Contains(value, "@") {
		v.AddError(
			field,
			"invalid email format",
		)
	}
}
