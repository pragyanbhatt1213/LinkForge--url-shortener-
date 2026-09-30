package main

import (
	"context"
	"log"
	"os"

	"linkforge-url-shortener/internal/handlers"
	"linkforge-url-shortener/internal/middleware"
	"linkforge-url-shortener/internal/services"
	"linkforge-url-shortener/internal/store"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// Connect to PostgreSQL
	db, err := store.NewPostgres()
	if err != nil {
		log.Fatal("failed to connect to PostgreSQL:", err)
	}
	defer db.Close(context.Background())

	store.SetPostgresDB(db)

	// Start background click workers
	services.StartClickWorkers(3)

	log.Println("PostgreSQL connected")

	// Create database tables
	err = store.CreateTables(db)
	if err != nil {
		log.Fatal("failed to create tables:", err)
	}

	log.Println("Database tables ready")

	app := fiber.New()

	// Initialize Prometheus metrics
	middleware.InitMetrics()

	// Record HTTP metrics
	app.Use(middleware.MetricsMiddleware())

	// Prometheus metrics endpoint
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Serve the dashboard at the root path.
	// All API routes registered after this still work normally.
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	})

	app.Post(
		"/shorten",
		middleware.RateLimit(store.GetRedisClient()),
		handlers.Shorten,
	)

	app.Get("/analytics/:slug", handlers.Analytics)
	app.Get("/links", handlers.ListLinks)
	app.Get("/:slug", handlers.Redirect)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(app.Listen(":" + port))
}
