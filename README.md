# StockFlow

StockFlow is an inventory application for managing products, recording stock movements, and
creating and cancelling sales orders. PostgreSQL stores the data; the frontend is a React
application backed by a Go API.

## Quick start

The easiest way to run the full app is with Docker Compose. The first run may take longer while
Docker downloads images and the frontend installs dependencies.

Prerequisite: Docker with the Docker Compose plugin.

### Run the full stack

From the repository root, run:

```powershell
docker compose up --build
```

Compose starts PostgreSQL, applies migrations, loads sample data, and runs the API and frontend.
Open [http://localhost:5173](http://localhost:5173). Press Ctrl+C to stop the services; the
PostgreSQL volume and its data are retained.

### Run the API and frontend from source

For local development with Go and Node.js, install Go 1.27 and Node.js 24 with npm. First start
PostgreSQL and load the sample data from the repository root:

```sh
docker compose up -d db migrate seed
docker compose logs migrate seed
```

Wait for the database to become healthy and for migration and seed to finish. Then run the API in
one terminal from the repository root.

PowerShell:

```powershell
Set-Location backend
$env:DATABASE_URL = "postgres://stockflow:stockflow@localhost:5432/stockflow?sslmode=disable"
go run ./cmd/api
```

Bash:

```sh
cd backend
export DATABASE_URL="postgres://stockflow:stockflow@localhost:5432/stockflow?sslmode=disable"
go run ./cmd/api
```

The API listens on `http://localhost:8080` by default. `DATABASE_URL` is required; `PORT` is
optional. In a second terminal, start the frontend:

```powershell
Set-Location frontend
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173). Vite proxies `/api` requests to the local
API, so no separate CORS configuration is needed for development. Check API readiness with:

```sh
curl -i http://localhost:8080/api/health
```

The expected response is `200 OK` with `{"status":"ok"}`.

## Tests

### Backend

For backend tests that do not need PostgreSQL, run from `backend`:

```sh
go test ./...
```

Database-backed tests are skipped if `TEST_DATABASE_URL` is not set. To run the full backend suite
with PostgreSQL and Go's race detector, run from the repository root:

```sh
docker compose run --rm backend-test
```

The test service runs in Docker because Go's `-race` detector requires cgo. It uses the separate
`stockflow_test` database and enables cgo. The test harness resets that database's `public` schema;
never point `TEST_DATABASE_URL` at a database containing data you need to keep.

### Frontend

Run from the repository root:

```sh
cd frontend
npm install
npm test
```

## Seeing the product list states

- **Loading:** open the products page on its first load. To see the state again, enable network
  throttling in your browser's developer tools and reload.
- **Empty:** search for `zzz`, which does not match the sample products.
- **Error and retry:** stop the API with Ctrl+C in its terminal, then change a product filter to
  trigger a request. The page shows an error and Retry button. Restart the API and click Retry.

## Managing products

Use **New product** to create a product with a unique SKU, name, price, and optional initial stock.
Use **Edit** in a product row to change its name, price, or active status; SKU is read-only and
stock changes are recorded with **Adjust stock**. **Delete** asks for confirmation and only removes
products without stock history. Products with history cannot be deleted; deactivate them instead.

## API

All endpoints are served from `http://localhost:8080`. Product and order lists are paginated:
`page` defaults to `1`, `limit` defaults to `20`, and the maximum limit is `100`.

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/health` **(added)** | Check API and database availability |
| GET | `/api/products` | List/search products; accepts `search`, `page`, and `limit`; `filter` is **added** |
| POST | `/api/products` | Create a product |
| PUT | `/api/products/{id}` | Update product name, price, or active status |
| DELETE | `/api/products/{id}` | Delete a product with no stock history |
| POST | `/api/products/{id}/stock` | Record a stock adjustment |
| GET | `/api/products/{id}/movements` | List a product's stock movements |
| POST | `/api/orders` | Create an order |
| GET | `/api/orders` **(added)** | List orders |
| GET | `/api/orders/{id}` | Get an order and its items |
| POST | `/api/orders/{id}/cancel` | Cancel an order and restore its stock |
| GET | `/api/summary` **(added)** | Get the product and stock summary |

The product `filter` accepts `all` (the default), `active`, `inactive`, or `low_stock`. The
low-stock filter includes active products with stock at or below 10. Creating a product accepts
`sku`, `name`, `price`, and optional `stock`/`is_active`. Updating a product accepts `name`,
`price`, and `is_active`; SKU is immutable. Order items contain only `sku` and `qty`; the server
supplies prices.

Errors use this JSON envelope (optional fields are omitted when not applicable):

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "validation failed",
    "details": [{ "field": "name", "message": "is required" }]
  }
}
```

| HTTP status | Error codes |
|---|---|
| 400 | `INVALID_JSON`, `VALIDATION_FAILED` |
| 404 | `NOT_FOUND` |
| 405 | `METHOD_NOT_ALLOWED` |
| 409 | `SKU_ALREADY_EXISTS`, `PRODUCT_HAS_HISTORY`, `INSUFFICIENT_STOCK`, `ORDER_ALREADY_CANCELLED` |
| 500 | `INTERNAL` |
| 503 | `UNAVAILABLE` |

`INSUFFICIENT_STOCK` includes the requested `sku` and current `available` quantity. The health
endpoint returns `UNAVAILABLE` if it cannot reach PostgreSQL.

The following examples use Bash syntax. On Windows, run them in Git Bash or use `curl.exe` in
PowerShell. The examples that create, update, adjust, or delete data change the local development
database. The order detail and cancellation examples assume a freshly seeded database, where the
first order created after the three sample orders has ID `4`; use the ID returned by your create
request if your database already contains other orders.

```sh
# List active products matching "beras".
# Returns 200 with a paged product response.
curl -s "http://localhost:8080/api/products?search=beras&filter=active&page=1&limit=20"

# Create a product (201). Repeating this request returns 409 SKU_ALREADY_EXISTS.
curl -i -X POST http://localhost:8080/api/products \
  -H 'Content-Type: application/json' \
  -d '{"sku":"SBN-MND-100G","name":"Sabun Mandi 100g","price":4500,"stock":24}'

# Update product 1 (200).
curl -i -X PUT http://localhost:8080/api/products/1 \
  -H 'Content-Type: application/json' \
  -d '{"name":"Beras Premium 5kg","price":79000,"is_active":true}'

# Attempt to remove more stock than product 9 has; returns 409 INSUFFICIENT_STOCK (available: 6).
curl -i -X POST http://localhost:8080/api/products/9/stock \
  -H 'Content-Type: application/json' \
  -d '{"type":"out","qty":999}'

# Add stock to product 9 (200).
curl -i -X POST http://localhost:8080/api/products/9/stock \
  -H 'Content-Type: application/json' \
  -d '{"type":"in","qty":24,"note":"supplier delivery"}'

# Read product 9's movement history.
# Returns 200 with a data array.
curl -s http://localhost:8080/api/products/9/movements

# Product 1 has stock history, so this returns 409 PRODUCT_HAS_HISTORY.
curl -i -X DELETE http://localhost:8080/api/products/1

# GAS-LPG-3KG (product 14 in the seed) has no history, so this returns 204.
curl -i -X DELETE http://localhost:8080/api/products/14

# Request too much KCP-MNS-600ML stock; returns 409 with sku and available quantity.
curl -i -X POST http://localhost:8080/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Toko Makmur","items":[{"sku":"BRS-PRM-5KG","qty":2},{"sku":"KCP-MNS-600ML","qty":50}]}'

# Create an order (201); on a fresh seed its total is 156000.
curl -i -X POST http://localhost:8080/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Toko Makmur","items":[{"sku":"BRS-PRM-5KG","qty":2}]}'

# List orders (200).
curl -s "http://localhost:8080/api/orders?page=1&limit=20"

# On a fresh seed, inspect and cancel the created order (ID 4); the second cancel returns 409.
# The detail and first cancellation return 200.
curl -s http://localhost:8080/api/orders/4
curl -s -X POST http://localhost:8080/api/orders/4/cancel
curl -i -X POST http://localhost:8080/api/orders/4/cancel

# Read the summary cards' data.
curl -s http://localhost:8080/api/summary
```

## Design decisions

- **Guarded stock updates:** order creation decrements stock with one `UPDATE ... WHERE stock >=
  qty` statement. At PostgreSQL's default `READ COMMITTED` isolation level, concurrent updates to
  the same row serialize on the row lock; after waiting, PostgreSQL rechecks the predicate against
  the latest row version. This prevents two orders from selling the same remaining units.
- **Consistent lock order:** products in a multi-item order are updated in ascending SKU order to
  reduce deadlock risk when concurrent orders contain the same products.
- **Database stock constraint:** `CHECK (stock >= 0)` is a final guard against negative stock,
  even if a future write path misses application validation.
- **Derived movement timestamp:** `last_movement_at` is selected from the most recent stock
  movement instead of stored on `products`, avoiding duplicated state that could become stale.
- **Movements preserve history:** order creation and cancellation both write stock movements
  linked to the order. The product-to-movement foreign key prevents deleting products with
  history.
- **Normalized, immutable SKU:** the API trims and uppercases SKUs on creation; update requests do
  not accept a SKU.
- **Wide order totals:** order totals and calculated line totals use `BIGINT` to avoid overflow
  when multiplying price by quantity.
- **Strict request JSON:** unknown fields, malformed bodies, and trailing JSON are rejected.
  Order prices are calculated from stored product prices rather than accepted from clients.
- **Vite API proxy:** the development server forwards `/api` requests to the backend, avoiding
  the need to configure browser CORS for local development.
- **TanStack Query invalidation:** successful product, stock, and order mutations invalidate the
  relevant product, summary, movement, or order queries so views refresh from the server.
- **Contract additions:** product filtering, `GET /api/orders`, `GET /api/summary`, and
  `GET /api/health` extend the core API contract.

## What I'd improve

- Add idempotency keys to order creation.
- Add authentication and an audit trail.
- Add a `pg_trgm` index if product search needs to scale.
- Add browser-level Playwright end-to-end tests.
- Generate an OpenAPI specification from the handlers.
- Run the race-enabled backend suite in CI.
- Consider optimistic UI updates where their failure behavior is clear.

## Hours spent

Time was not tracked per phase, so reliable hour totals are unavailable.

| Phase | Time |
|---|---|
| Planning and design | Not tracked |
| Backend | Not tracked |
| Frontend | Not tracked |
| Documentation and verification | Not tracked |

## Sample data and database reset

[`backend/seed/seed.sql`](backend/seed/seed.sql) loads 15 products, their initial stock
movements, and three example orders, including one cancelled order. The seed skips all data if
the `products` table already contains rows. PostgreSQL data is kept in a persistent Docker volume,
so restarting Compose does not reset or reload the sample data.

To delete the local database volume and load the sample data again:

```sh
docker compose down -v
docker compose up -d
```

> **Warning:** `docker compose down -v` permanently deletes the local PostgreSQL volume and all
> data stored in it.
