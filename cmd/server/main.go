package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"go-tecdoc-api/internal/auth"
	"go-tecdoc-api/internal/database"
	"go-tecdoc-api/internal/handlers"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Database connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&interpolateParams=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("✓ Database connection established")

	// Initialize database queries
	queries := database.New(db)

	// Initialize handlers
	h := handlers.New(queries)

	// Setup router
	router := mux.NewRouter()
	router.Use(recoverMiddleware)

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(auth.NewAPIKeyMiddleware(queries).Middleware)
	api.Use(h.ValidateCommonQueryMiddleware)

	registerAPIRoutes(api, h)

	// Health check - actually pings the database instead of assuming it is
	// still reachable, so orchestrators (k8s/Docker) can detect a dead DB
	// connection and restart/stop routing traffic to this instance.
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"error","database":"unreachable"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok","database":"connected"}`)
	}).Methods("GET")

	// CORS. Wildcard origins combined with AllowCredentials is both invalid
	// per the fetch/CORS spec and unsafe (browsers will either reject it or,
	// depending on the library, effectively echo back "*" as an allowed
	// origin for credentialed requests). This API is authenticated via the
	// X-API-Key header rather than cookies, so credentials are not required;
	// ALLOWED_ORIGINS lets deployments restrict origins explicitly when the
	// storefront/admin call this API directly from the browser.
	allowedOrigins := []string{"*"}
	if raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")); raw != "" {
		allowedOrigins = nil
		for _, origin := range strings.Split(raw, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				allowedOrigins = append(allowedOrigins, origin)
			}
		}
	}
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-API-Key"},
		AllowCredentials: false,
		MaxAge:           300,
	})

	// Get server configuration
	serverHost := os.Getenv("SERVER_HOST")
	if serverHost == "" {
		serverHost = "0.0.0.0"
	}
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8081"
	}

	address := fmt.Sprintf("%s:%s", serverHost, serverPort)

	// A bare http.ListenAndServe has no read/write/idle timeouts, so a slow
	// or malicious client can hold a connection (and, transitively, a slot
	// in the DB pool once a handler starts a query) open indefinitely.
	// searchQueryTimeout bounds the slowest legitimate handler, so the HTTP
	// timeouts are set with headroom above it.
	server := &http.Server{
		Addr:              address,
		Handler:           corsHandler.Handler(router),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	log.Printf("🚀 Server starting on http://%s", address)
	log.Printf("📚 API documentation: http://%s/api/v1", address)
	log.Printf("💚 Health check: http://%s/health", address)
	log.Println("\n📋 API groups: localization, suppliers, vehicles, articles, catalog, search, enrichment, datasets, exports")
	log.Println("   Discovery: GET /api/v1")
	log.Println("   KBA:       GET /api/v1/search/kba?number=...")
	log.Println("   Datasets:  GET /api/v1/datasets")
	log.Println("   Reference: API_REFERENCE.md / OPENAPI.yaml")

	// Run the server in the background and shut it down gracefully on
	// SIGINT/SIGTERM so in-flight requests finish instead of being dropped
	// mid-response on every deploy/restart.
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	case sig := <-stop:
		log.Printf("Received %s, shutting down gracefully...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed, forcing close: %v", err)
			_ = server.Close()
		}
		log.Println("Server stopped")
	}
}

// recoverMiddleware converts a panic in any handler into a 500 response
// instead of crashing the whole process and dropping every other in-flight
// request. Handlers here do a lot of manual SQL scanning; a single bad
// assumption (unexpected NULL, index-out-of-range) should not take the API
// down for every other API key.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v", rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":"internal_error","message":"Unexpected server error"}`)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
