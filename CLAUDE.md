# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Run the application
go run cmd/api/main.go

# Build the binary
go build -o bin/api cmd/api/main.go

# Run tests
go test ./...

# Run a single test
go test -run TestName ./path/to/package

# Format code
go fmt ./...

# Vet code
go vet ./...

# Run database migration
mysql -u root -p db_go_basics < migrations/001_create_users_table.sql
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `DB_DSN` | MySQL connection string | `root:root@tcp(localhost:3306)/db_go_basics?parseTime=true` |
| `JWT_SECRET` | Secret key for JWT signing | (development default) |
| `JWT_ACCESS_TOKEN_DURATION` | Token validity duration | `15m` |

## Architecture

This is a Go REST API using **Clean Architecture** with MySQL database and JWT authentication.

### Layer Structure

```
cmd/api/              → Application entrypoint
config/               → Configuration management (env vars)
internal/
  app/                → Server bootstrap and dependency wiring
  auth/               → JWT token handling and middleware
  domain/user/        → Domain layer: entity, repository interface, service, errors
  repository/mysql/   → MySQL implementation of repository interface
  handler/http/       → HTTP handlers (Go 1.22+ routing)
migrations/           → SQL migration files
```

### Dependency Flow

```
HTTP Handler → Service → Repository Interface ← MySQL Repository
     ↓
JWT Middleware (for protected routes)
```

### API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/register` | No | Create new user |
| POST | `/login` | No | Authenticate and get JWT |
| GET | `/me` | Yes | Get current user |
| GET | `/users/{id}` | Yes | Get user by ID |
| PUT | `/users/{id}` | Yes | Update user (own profile only) |
| DELETE | `/users/{id}` | Yes | Soft-delete user (own account only) |
| GET | `/health` | No | Health check |

### Adding a New Domain Entity

1. Create `internal/domain/{entity}/entity.go` - Define the struct
2. Create `internal/domain/{entity}/repository.go` - Define repository interface
3. Create `internal/domain/{entity}/errors.go` - Define domain errors
4. Create `internal/domain/{entity}/service.go` - Implement business logic
5. Create `internal/repository/mysql/{entity}_repository.go` - MySQL implementation
6. Create `internal/handler/http/{entity}_handler.go` - HTTP handlers
7. Wire dependencies in `internal/app/server.go`
8. Add migration in `migrations/`
