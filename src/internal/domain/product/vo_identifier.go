package product

import (
	"strings"

	"github.com/google/uuid"
)

func NewIdentifier(id string) (string, error) {
	id = strings.TrimSpace(id)
	if _, err := uuid.Parse(id); err != nil {
		return "", ErrProductIDInvalid
	}
	return id, nil
}
