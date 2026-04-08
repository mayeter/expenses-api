package main

import (
	"fmt"
	"log"

	"expenses-backend/internal/config"
	"expenses-backend/internal/db"
	"expenses-backend/internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.Load()

	gdb, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := db.Migrate(gdb); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := db.SeedCategories(gdb); err != nil {
		log.Fatalf("seed: %v", err)
	}

	app := fiber.New(fiber.Config{
		AppName: "expenses-api",
	})
	app.Use(logger.New())

	handlers.Register(app, gdb, cfg.AllowedOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatal(err)
	}
}
