package main

import (
	"log"
	"os"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/config"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/rest"
)

func main() {
	postgres, err := config.InitPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer postgres.Close()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET env var is required")
	}

	rest.NewRouter(postgres, jwtSecret).Server.Run(":8082")
}
