# Go Basics - REST API with Clean Architecture

A learning project demonstrating Go best practices: Clean Architecture, JWT authentication, and CRUD operations with MySQL.

## Prerequisites

- Go 1.22+ (for enhanced routing features)
- MySQL 8.0+

## Setup

### 1. Clone and Install Dependencies

```bash
git clone https://github.com/yourusername/go-basics.git
cd go-basics
go mod tidy
```

### 2. Create Database and Run Migration

```bash
# Login to MySQL
mysql -u root -p

# Run the migration (from MySQL prompt)
source migrations/001_create_users_table.sql

# Or run directly from terminal
mysql -u root -p < migrations/001_create_users_table.sql
```

### 3. Configure Environment (Optional)

The app works with defaults, but you can customize:

```bash
export DB_DSN="root:yourpassword@tcp(localhost:3306)/db_go_basics?parseTime=true"
export JWT_SECRET="your-secret-key-min-32-chars-long"
export SERVER_PORT="8080"
```

### 4. Run the Server

```bash
go run cmd/api/main.go
```

You should see:
```
Configuration loaded
Database connection established
HTTP server listening on :8080
```

## API Testing

### Health Check

```bash
curl http://localhost:8080/health
```

Response: `ok`

### Register a New User

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

Response:
```json
{"id":1,"email":"test@example.com"}
```

### Login (Get JWT Token)

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {"id":1,"email":"test@example.com"}
}
```

**Save the token** for authenticated requests below.

### Get Current User (Protected)

```bash
curl http://localhost:8080/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

Response:
```json
{"id":1,"email":"test@example.com"}
```

### Get User by ID (Protected)

```bash
curl http://localhost:8080/users/1 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### Update User (Protected - Own Profile Only)

```bash
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "email": "newemail@example.com",
    "password": "newpassword123"
  }'
```

### Delete User (Protected - Own Account Only)

```bash
curl -X DELETE http://localhost:8080/users/1 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

Response: `204 No Content`

## Project Structure

```
go-basics/
├── cmd/api/                  # Application entrypoint
├── config/                   # Configuration management
├── internal/
│   ├── app/                  # Server bootstrap & dependency wiring
│   ├── auth/                 # JWT & middleware
│   ├── domain/user/          # Business logic & entities
│   ├── handler/http/         # HTTP handlers
│   └── repository/mysql/     # Database layer
└── migrations/               # SQL migrations
```

## Error Responses

All errors return JSON:

```json
{"error": "error message here"}
```

| Status | Meaning |
|--------|---------|
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Invalid/missing token |
| 403 | Forbidden - Not allowed (e.g., updating other user) |
| 404 | Not Found - User doesn't exist |
| 409 | Conflict - Email already exists |
| 500 | Internal Server Error |

## License

MIT
