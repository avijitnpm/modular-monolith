package billing

import "errors"

var ErrInvalidTransition = errors.New("invalid subscription status transition")

var allowedTransitions = map[string]map[string]bool{
	"trialing":  {"active": true, "cancelled": true},
	"active":    {"past_due": true, "cancelled": true},
	"past_due":  {"active": true, "cancelled": true},
	"cancelled": {"expired": true},
	"expired":   {},
}

func ValidateTransition(current string, next string) error {
	if current == next {
		return nil
	}

	targets, ok := allowedTransitions[current]
	if !ok {
		return ErrInvalidTransition
	}

	if !targets[next] {
		return ErrInvalidTransition
	}

	return nil
}
