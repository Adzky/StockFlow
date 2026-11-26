# StockFlow

StockFlow is a simple inventory application for managing products, adjusting stock, and recording sales orders. Docker Compose starts PostgreSQL, applies the database migrations, and loads sample data.

## Quick start

Run from the repository root with Docker Compose installed:

```sh
docker compose up -d
```

Check migration and seed startup:

```sh
docker compose logs migrate seed
```

Inspect the sample products:

```sh
docker compose exec db psql -U stockflow -d stockflow -c "SELECT id, sku, name, price, stock, is_active FROM products ORDER BY id;"
```

## Backend API

The API runs locally and connects to the PostgreSQL service started by Docker Compose. Start the database and migrations first:

```sh
docker compose up -d
```

In a PowerShell terminal, start the API from the `backend` directory:

```powershell
cd backend
$env:DATABASE_URL = "postgres://stockflow:stockflow@localhost:5432/stockflow?sslmode=disable"
go run ./cmd/api
```

`DATABASE_URL` is required. `PORT` is optional and defaults to `8080`. Check the health endpoint:

```powershell
curl.exe -i http://localhost:8080/api/health
```

The expected response is `200 OK` with `{"status":"ok"}`. Press Ctrl+C to shut down the API gracefully.

## Backend tests

Run the backend unit tests locally from the `backend` directory:

```sh
go test ./...
```

Database-backed server tests are skipped when `TEST_DATABASE_URL` is not set. To run the full suite, including those tests with Go's race detector, run this from the repository root:

```sh
docker compose run --rm backend-test
```

The test harness resets the `public` schema in the database named by `TEST_DATABASE_URL`. The Compose test service uses the separate `stockflow_test` database; do not point `TEST_DATABASE_URL` at a database containing data you need to keep. The test database is created by `backend/db/init/01-test-db.sql` when PostgreSQL initializes a new data volume.

## Sample data

[`backend/seed/seed.sql`](backend/seed/seed.sql) loads 15 products, their initial stock movements, and three example orders, including one cancelled order. The seed skips all data if the `products` table already contains any rows. PostgreSQL data is kept in a persistent Docker volume, so restarting Compose does not reset or reload the sample data.

To reset the local database and load the sample data again:

```sh
docker compose down -v
docker compose up -d
```

> **Warning:** `docker compose down -v` permanently deletes the local PostgreSQL volume and all data stored in it.