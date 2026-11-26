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

## Sample data

[`backend/seed/seed.sql`](backend/seed/seed.sql) loads 15 products, their initial stock movements, and three example orders, including one cancelled order. The seed skips all data if the `products` table already contains any rows. PostgreSQL data is kept in a persistent Docker volume, so restarting Compose does not reset or reload the sample data.

To reset the local database and load the sample data again:

```sh
docker compose down -v
docker compose up -d
```

> **Warning:** `docker compose down -v` permanently deletes the local PostgreSQL volume and all data stored in it.