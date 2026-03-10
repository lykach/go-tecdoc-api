package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

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

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()
	
    // Localization routes
    api.HandleFunc("/languages", h.GetLanguages).Methods("GET")
    api.HandleFunc("/languages/{id}", h.GetLanguageByID).Methods("GET")
    api.HandleFunc("/countries", h.GetCountries).Methods("GET")
    api.HandleFunc("/countries/{id}", h.GetCountryByID).Methods("GET")
	
	// Suppliers routes
    api.HandleFunc("/suppliers", h.GetSuppliers).Methods("GET")
    api.HandleFunc("/suppliers/{id}", h.GetSupplierByID).Methods("GET")
	api.HandleFunc("/suppliers/{id}/products", h.GetSupplierProducts).Methods("GET")

	// Manufacturers routes
	api.HandleFunc("/manufacturers", h.GetManufacturers).Methods("GET")
	api.HandleFunc("/manufacturers/{id}", h.GetManufacturerByID).Methods("GET")

	// Model series routes
	api.HandleFunc("/manufacturers/{id}/models", h.GetModelSeries).Methods("GET")
	api.HandleFunc("/models/{id}", h.GetModelSeriesDetails).Methods("GET")
	
	// Commercial Vehicles
    api.HandleFunc("/models/{id}/cv", h.GetCommercialVehicles).Methods("GET")
	api.HandleFunc("/cv/{id}", h.GetCommercialVehicleDetails).Methods("GET")
	
	// Motorcycles
    api.HandleFunc("/models/{id}/mc", h.GetMotorcycles).Methods("GET")
	api.HandleFunc("/mc/{id}", h.GetMotorcycleDetails).Methods("GET")

	// Passenger cars routes
	api.HandleFunc("/models/{id}/cars", h.GetPassengerCars).Methods("GET")
	api.HandleFunc("/cars/{id}", h.GetCarDetails).Methods("GET")
	api.HandleFunc("/cars/{id}/product-groups", h.GetCarProductGroups).Methods("GET")
	
	// Engine details
    api.HandleFunc("/engines/{id}", h.GetEngineDetails).Methods("GET")
	
	// Articles routes
	api.HandleFunc("/articles/search", h.SearchArticles).Methods("GET")
	api.HandleFunc("/articles/{id}", h.GetArticleDetails).Methods("GET")
	api.HandleFunc("/articles/{id}/applicability", h.GetArticleApplicability).Methods("GET")
	api.HandleFunc("/articles/{id}/cross-references", h.GetArticleCrossReferences).Methods("GET")
	api.HandleFunc("/articles/{id}/media", h.GetArticleMedia).Methods("GET")
	api.HandleFunc("/articles/{id}/components", h.GetArticleComponents).Methods("GET")
	api.HandleFunc("/articles/{id}/accessories", h.GetArticleAccessories).Methods("GET")
	api.HandleFunc("/articles/{id}/oem", h.GetArticleOEMNumbers).Methods("GET")
	api.HandleFunc("/articles/{id}/coordinates", h.GetArticleCoordinates).Methods("GET")
	api.HandleFunc("/articles/{id}/criteria", h.GetArticleCriteria).Methods("GET")

	// Product groups routes
	api.HandleFunc("/product-groups", h.GetProductGroups).Methods("GET")
	api.HandleFunc("/product-groups/{id}/children", h.GetCategoryChildren).Methods("GET")
	api.HandleFunc("/product-groups/{id}/articles", h.GetProductGroupArticles).Methods("GET")

	// Search routes
	api.HandleFunc("/search/kba", h.SearchByKBA).Methods("GET")
	api.HandleFunc("/search/article", h.SearchArticleByNumber).Methods("GET")
    api.HandleFunc("/search/article", h.SearchArticles).Methods("GET")
    api.HandleFunc("/search/oem", h.SearchByOEM).Methods("GET")
	api.HandleFunc("/search/analog", h.SearchAnalogs).Methods("GET")
	api.HandleFunc("/search/oem-oem", h.SearchOEMByOEM).Methods("GET")
	
	
	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok","database":"connected"}`)
	}).Methods("GET")

	// Setup CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
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

	// Start server
	log.Printf("🚀 Server starting on http://%s", address)
	log.Printf("📚 API documentation: http://%s/api/v1", address)
	log.Printf("💚 Health check: http://%s/health", address)
	log.Println("\n📋 Available endpoints:")
	log.Println("   GET /api/v1/languages")
    log.Println("   GET /api/v1/languages/{id}")
    log.Println("   GET /api/v1/countries")
    log.Println("   GET /api/v1/countries/{id}")
	log.Println("   GET /api/v1/suppliers")
    log.Println("   GET /api/v1/suppliers/{id}")
    log.Println("   GET /api/v1/suppliers?brand=BOSCH")
	log.Println("   GET /api/v1/suppliers/{id}/products")
	log.Println("   GET /api/v1/manufacturers")
	log.Println("   GET /api/v1/manufacturers/{id}")
	log.Println("   GET /api/v1/manufacturers/{id}/models")
	log.Println("   GET /api/v1/models/{id}")
	log.Println("   GET /api/v1/models/{id}/cars")
	log.Println("   GET /api/v1/cars/{id}")
	log.Println("   GET /api/v1/cars/{id}/product-groups")
	log.Println("   GET /api/v1/models/{id}/cv")
	log.Println("   GET /api/v1/cv/{id}")
	log.Println("   GET /api/v1/models/{id}/mc")
	log.Println("   GET /api/v1/mc/{id}")
	log.Println("   GET /api/v1/engines/{id}")
	log.Println("   GET /api/v1/articles/search?number=...")
	log.Println("   GET /api/v1/articles/{id}")
	log.Println("   GET /api/v1/articles/{id}/cross-references")
	log.Println("   GET /api/v1/articles/{id}/media")
	log.Println("   GET /api/v1/articles/{id}/components")
	log.Println("   GET /api/v1/articles/{id}/accessories")
	log.Println("   GET /api/v1/articles/{id}/oem")
	log.Println("   GET /api/v1/articles/{id}/coordinates")
	log.Println("   GET /api/v1/articles/{id}/criteria")
	log.Println("   GET /api/v1/search/article")
	log.Println("   GET /api/v1/search/oem")
	log.Println("   GET /api/v1/search/oem-oem")
	log.Println("   GET /api/v1/search/kba (not implemented)")
	log.Println("   GET /api/v1/search/analog")
	log.Println("   GET /api/v1/product-groups")
	log.Println("   GET /api/v1/product-groups/{id}/children")
	log.Println("   GET /api/v1/product-groups/{id}/articles?car_id=...")

	if err := http.ListenAndServe(address, corsHandler.Handler(router)); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}