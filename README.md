# HireMeMaybe Backend

Backend service for HireMeMaybe - A web application platform connecting Computer Engineering and Software and Knowledge Engineering students at Kasetsart University with employment opportunities.

## Requirements

- Go v1.24.6 or higher
- Docker & Docker Compose
- Make (for using Makefile commands)

## Quick Start

### 1. Clone the Repository
```bash
git clone https://github.com/HireMeMaybe/HireMeMaybe-backend.git
cd HireMeMaybe-backend
```

### 2. Setup Environment Variables

Copy the sample environment file and configure it:

Copy the sample environment file and configure it:

**macOS/Linux:**
```bash
cp sample.env .env
```

**Windows:**
```cmd
copy sample.env .env
```

**Required configurations:**
- `CPSK_GOOGLE_AUTH_CLIENT` - Google OAuth Client ID
- `CPSK_GOOGLE_AUTH_SECRET` - Google OAuth Client Secret
- Other variables can be left as default for local development

### 3. Start the Database

Start the PostgreSQL database container:
```bash
make docker-run
```

### 4. Install Dependencies

Download and install Go dependencies:
```bash
go mod tidy
```

### 5. Run the Server

Start the development server:
```bash
make run
```

The server will start at `http://localhost:8080`

**API Documentation:** Available at `http://localhost:8080/swagger/index.html`

## Available Commands

### Development

| Command | Description |
|---------|-------------|
| `make run` | Run the application |
| `make watch` | Run with live reload (auto-restart on changes) |
| `make build` | Build the application binary |
| `make clean` | Remove build artifacts |

### Database

| Command | Description |
|---------|-------------|
| `make docker-run` | Start PostgreSQL container |
| `make docker-down` | Stop PostgreSQL container |

### Testing

| Command | Description |
|---------|-------------|
| `make test` | Run all unit tests |
| `make itest` | Run integration tests with database |
| `make all` | Build and run tests |

## Project Structure

```
HireMeMaybe-backend/
├── cmd/                    # Application entry points
│   ├── api/               # Main API server
│   ├── create-admin/      # Admin user creation tool
│   └── clean-db/          # Database cleanup utility
├── internal/              # Private application code
│   ├── auth/             # Authentication & authorization
│   ├── controller/       # HTTP request handlers
│   ├── database/         # Database configuration
│   ├── middleware/       # HTTP middleware (auth, CORS, rate limiting)
│   ├── model/            # Data models
│   ├── server/           # Server setup and routes
│   └── utilities/        # Helper functions
├── docs/                 # Swagger API documentation
├── .env                  # Environment variables (create from sample.env)
├── go.mod                # Go module dependencies
└── Makefile              # Development commands
```

## Security Features

- **JWT Authentication** - Token-based authentication with blacklist support
- **OAuth 2.0** - Google OAuth integration for CPSK users
- **Role-Based Access Control** - Admin, Company, CPSK, and Visitor roles
- **Rate Limiting** - Protection against brute force attacks
- **Security Headers** - HSTS, X-Frame-Options, X-Content-Type-Options
- **Input Validation** - Request validation and sanitization
- **File Upload Security** - Size limits, extension validation, cloud storage

## Environment Variables

Key environment variables (see `sample.env` for complete list):

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | `localhost:5432` |
| `JWT_SECRET` | Secret key for JWT signing | - |
| `CPSK_GOOGLE_AUTH_CLIENT` | Google OAuth Client ID | - |
| `CPSK_GOOGLE_AUTH_SECRET` | Google OAuth Client Secret | - |
| `ALLOW_ORIGIN` | CORS allowed origins (comma-separated) | `http://localhost:3000` |
| `CLOUD_STORAGE_BUCKET` | Cloud storage bucket name | - |

## Running Tests

### Unit Tests
```bash
make test
```

### Integration Tests (requires database)
```bash
make itest
```

### Test Coverage
```bash
go test ./... -cover
```

## API Documentation

Once the server is running, access the interactive Swagger documentation at:

```
http://localhost:8080/swagger/index.html
```
