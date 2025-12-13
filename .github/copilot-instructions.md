# Trawl - Distributed Search Engine

## Project Architecture

This is a microservices-based distributed search engine (like Elasticsearch) with 4 independent Go services orchestrated via Go workspaces. Each service has its own `go.mod` and can be developed/deployed independently.

**Services:**

- `services/auth` - JWT authentication, user management (PostgreSQL + sqlc)
- `services/search` - Query processing, TF-IDF ranking, result merging
- `services/indexing` - Document parsing, tokenization, inverted index building (async via RabbitMQ)
- `services/shared` - Common utilities, types, and helpers

**Key Infrastructure:**

- PostgreSQL for auth/user data
- ScyllaDB for inverted index storage (distributed, sharded)
- RabbitMQ for async job queues (indexing, analytics)
- MinIO (S3-compatible) for document storage
- Nginx for load balancing and routing

## Critical Patterns

### Go Workspace Structure

The project uses Go workspaces (`go.work`) to manage multiple modules. Always run commands from service directories:

```bash
cd services/auth
go run cmd/api/main.go
```

### Database Code Generation with sqlc

We use **sqlc** to generate type-safe Go code from SQL. Never manually edit files in `internal/db/` - they're auto-generated.

**Workflow:**

1. Define schema in `sql/schema/schema.sql`
2. Write queries in `sql/queries/*.sql` with sqlc annotations (e.g., `-- name: GetUserByEmail :one`)
3. Run `sqlc generate` to regenerate Go code
4. Import generated code: `import "github.com/amrrdev/trawl/services/auth/internal/db"`

**Config location:** `services/auth/sqlc.yaml` (specifies PostgreSQL, pgx/v5, JSON tags)

### Database Migrations

Use golang-migrate with custom migration runner at [services/auth/cmd/migrate/main.go](services/auth/cmd/migrate/main.go):

```bash
# Apply migrations
go run cmd/migrate/main.go -direction=up

# Rollback N migrations
go run cmd/migrate/main.go -direction=down -steps=1
```

Migration files live in `internal/database/migrations/` with pairs: `000001_name.up.sql` and `000001_name.down.sql`.

### Configuration Management

Each service loads config from `.env` (2 levels up from cmd) via godotenv. See [services/auth/internal/config/config.go](services/auth/internal/config/config.go) for the pattern:

- Attempts to load `.env` from `../../.env` (relative to cmd directory)
- Falls back to hardcoded defaults if `.env` missing
- Never commit `.env` to version control

### Database Connection Pattern

Standard pattern in [services/auth/internal/database/database.go](services/auth/internal/database/database.go):

1. Parse connection URL with `pgxpool.ParseConfig()`
2. Apply custom config (max/min conns, timeouts, health check period)
3. Create pool with `pgxpool.NewWithConfig()`
4. Always ping before returning
5. Expose `HealthCheck()` and `Stats()` methods

**Default pool settings:**

- MaxConns: 25, MinConns: 5
- MaxConnLifetime: 1 hour, MaxConnIdleTime: 30 min
- HealthCheckPeriod: 1 minute

## Development Workflows

### Running Services Locally

1. Start infrastructure: `docker-compose up -d` (starts PostgreSQL)
2. Run migrations: `cd services/auth && go run cmd/migrate/main.go -direction=up`
3. Start service: `go run cmd/api/main.go`

### Database Access

Connect to PostgreSQL container:

```bash
docker exec -it search-flow psql -U search-flow_user -d search-flow-db
```

### Module Organization

- `cmd/` - Entry points (main.go files for api, migrate, workers)
- `internal/` - Private implementation (config, database, handlers, business logic)
- `sql/` - Schema and queries for sqlc (auth service only)
- Use `github.com/amrrdev/trawl/services/{service-name}` import paths

## Key Concepts

**Inverted Index:** Core data structure mapping `word → [document IDs containing word]`. Enables O(log n) search instead of O(n) sequential scan.

**Async Processing:** Documents uploaded to indexing service return immediately (202 Accepted). Actual indexing happens via RabbitMQ workers to avoid blocking API.

**Sharding Strategy:** ScyllaDB distributes inverted index across nodes. Query coordinator merges results from multiple shards.

## Common Tasks

**Add a new database query:**

1. Add SQL to `services/auth/sql/queries/*.sql` with sqlc comment
2. Run `sqlc generate` from auth directory
3. Use generated methods: `queries.GetUserByEmail(ctx, email)`

**Create a new migration:**

1. Create numbered pair in `services/auth/internal/database/migrations/`
2. Name format: `000002_description.up.sql` / `000002_description.down.sql`
3. Run with migrate command

**Add a new service:**

1. Create directory under `services/`
2. Add `go.mod` with `module github.com/amrrdev/trawl/services/{name}`
3. Update `go.work` file with `use ./services/{name}`
