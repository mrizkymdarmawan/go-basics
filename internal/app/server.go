// Package app handles application bootstrap and dependency wiring.
// This is the "composition root" - where all dependencies come together.
//
// COMPOSITION ROOT PATTERN:
// The composition root is the ONE place in your application where:
// 1. All dependencies are created
// 2. Dependencies are "wired" (connected) together
// 3. The application is started
//
// Benefits:
// - Single place to see all dependencies
// - Easy to change implementations (e.g., swap MySQL for PostgreSQL)
// - Keeps the rest of the code decoupled
package app

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	// Import MySQL driver
	// The underscore (_) means we import for side effects only.
	// The driver registers itself with database/sql when imported.
	_ "github.com/go-sql-driver/mysql"

	"go-basics/config"
	"go-basics/internal/auth"
	"go-basics/internal/domain/user"
	userHandler "go-basics/internal/handler/http"
	userRepo "go-basics/internal/repository/mysql"
)

// Run starts the application.
// This is the main entry point called from cmd/api/main.go.
//
// The function:
// 1. Loads configuration
// 2. Connects to the database
// 3. Creates all dependencies
// 4. Starts the HTTP server
func Run() error {
	// Step 1: Load configuration
	// Configuration is loaded from environment variables with defaults.
	cfg := config.Load()
	log.Println("Configuration loaded")

	// Step 2: Connect to database
	db, err := openDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	// defer ensures db.Close() is called when Run() returns.
	// This is important for cleaning up database connections.
	defer db.Close()
	log.Println("Database connection established")

	// Step 3: Create dependencies (Dependency Injection)
	// We create dependencies in order: lowest level first.
	//
	// Dependency graph:
	//   UserRepository (database) <-- used by
	//   UserService (business logic) <-- used by
	//   UserHandler (HTTP) <-- used by
	//   HTTP Server

	// Repository layer - data access
	userRepository := userRepo.NewUserRepository(db)

	// Service layer - business logic
	userService := user.NewService(userRepository)

	// Auth components
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.Issuer,
	)
	authMiddleware := auth.NewMiddleware(jwtManager)

	// Handler layer - HTTP
	userHTTPHandler := userHandler.NewUserHandler(userService, jwtManager)

	// Step 4: Set up HTTP routing
	mux := http.NewServeMux()

	// Health check endpoint
	// This is used by load balancers and container orchestrators
	// to check if the application is running.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Register user routes
	userHTTPHandler.RegisterRoutes(mux, authMiddleware)

	// Step 5: Configure and start HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,

		// Timeouts prevent slow clients from holding connections.
		// These are important for security and resource management.
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Printf("HTTP server listening on :%s", cfg.Server.Port)

	// ListenAndServe blocks until the server shuts down.
	// It returns an error if the server fails to start.
	return server.ListenAndServe()
}

// openDB creates a database connection pool.
//
// IMPORTANT: *sql.DB is a connection POOL, not a single connection.
// - It manages multiple connections automatically
// - It's safe for concurrent use from multiple goroutines
// - You should create ONE *sql.DB per database and reuse it
// - Don't call db.Close() until the application shuts down
func openDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	// sql.Open doesn't actually connect to the database.
	// It just validates the DSN and prepares the pool.
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Configure the connection pool
	//
	// MaxOpenConns: Maximum number of open connections.
	// Too high = exhausts database resources.
	// Too low = connection contention under load.
	// Start with 10-25 and tune based on load testing.
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	// MaxIdleConns: Maximum idle connections in the pool.
	// Idle connections are kept open for reuse.
	// Should be <= MaxOpenConns.
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	// ConnMaxLifetime: How long a connection can be reused.
	// Helps with:
	// - Load balancing (new connections go to new servers)
	// - Handling database restarts
	// - Preventing stale connections
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Ping actually connects to verify the configuration.
	// This is where you'll see errors like "connection refused".
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return db, nil
}
