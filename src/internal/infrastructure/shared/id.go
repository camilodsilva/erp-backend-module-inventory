package shared

import (
	"strings"

	"github.com/google/uuid"
)

func GenerateID() string {
	id, _ := uuid.NewV7()
	return id.String()
}

type UUIDGenerator struct{}

func (g *UUIDGenerator) Generate() string {
	return GenerateID()
}

func SchemaName(tenantID string) string {
	return "t_" + strings.ReplaceAll(tenantID, "-", "")
}
