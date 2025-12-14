package app

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go-basics/internal/domain/user"
	userHandler "go-basics/internal/handler/http"
	userRepo "go-basics/internal/repository/mysql"
)

func openDB() (*sql.DB, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		// Example fallback (development only)
		dsn = "root:root@tcp(localhost:3306)/db_go_basics?parseTime=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func Run() error {
	db, err := openDB()
	if err != nil {
		return err
	}

	// Dependency wiring
	userRepository := userRepo.NewUserRepository(db)
	userService := user.NewService(userRepository)
	userHTTPHandler := userHandler.NewUserHandler(userService)

	// HTTP routing (net/http – standard library)
	mux := http.NewServeMux()

	// Placeholder route (will expand next step)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Register routes
	userHTTPHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("HTTP server listening on :8080")
	return server.ListenAndServe()
}
