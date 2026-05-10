package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	"go-tecdoc-api/internal/database"
)

func main() {
	name := flag.String("name", "", "API key name, required")
	ownerName := flag.String("owner-name", "", "Owner name")
	ownerEmail := flag.String("owner-email", "", "Owner email")
	expires := flag.String("expires", "", "Expiration date in YYYY-MM-DD format, empty means no expiration")
	allowedIPs := flag.String("allowed-ips", "", "Comma-separated allowed IP addresses or CIDR ranges")
	allowedRoutes := flag.String("allowed-routes", "", "Comma-separated allowed route prefixes, for example /api/v1/articles/*,/api/v1/search/*")
	rateLimit := flag.Int("rate-limit", 120, "Requests per minute")
	flag.Parse()

	if strings.TrimSpace(*name) == "" {
		log.Fatal("-name is required")
	}

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&interpolateParams=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	var expiresAt *time.Time
	if strings.TrimSpace(*expires) != "" {
		parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(*expires), time.Local)
		if err != nil {
			log.Fatalf("invalid -expires value: %v", err)
		}
		endOfDay := parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		expiresAt = &endOfDay
	}

	queries := database.New(db)
	created, err := queries.CreateAPIKey(database.CreateAPIKeyParams{
		Name:               *name,
		OwnerName:          *ownerName,
		OwnerEmail:         *ownerEmail,
		ExpiresAt:          expiresAt,
		AllowedIPs:         *allowedIPs,
		AllowedRoutes:      *allowedRoutes,
		RateLimitPerMinute: *rateLimit,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("API key created successfully.")
	fmt.Println("Save this key now. It will not be stored in plain text:")
	fmt.Println(created.Plain)
	fmt.Printf("ID: %d\n", created.Record.ID)
	fmt.Printf("Prefix: %s\n", created.Record.KeyPrefix)
}
