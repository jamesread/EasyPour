# Database migrations

EasyPour applies [sql-migrate](https://github.com/rubenv/sql-migrate) upgrades before the service starts (container entrypoint) or via `make` in development. The binary does not create tables itself; it checks that the required migration id is present in the `migrations` table and refuses to start if it is missing.

## Drivers

EasyPour ships SQLite only:

```
database/sqlite/
```

## Development

Set `DB_PATH` to the SQLite file (absolute path preferred), then:

```bash
export DB_PATH=/path/to/easypour.db
make migrate
# or
make -C database/sqlite
```

Useful targets in `database/sqlite/`: `default` (up), `down`, `status`.

## Runtime (Docker)

The entrypoint runs `sql-migrate up` in `/var/app/database/sqlite`, then starts the service.

| Env | Role |
|-----|------|
| `DB_DRIVER` | Defaults to `sqlite` |
| `DB_PATH` | SQLite file path; defaults to `$EASYPOUR_CONFIG_FILE`’s directory + `easypour.db`, or `/config/easypour.db` |
| `EASYPOUR_CONFIG_FILE` | Config path (also used to derive default `DB_PATH`) |

Manual run inside a container:

```bash
docker exec -it easypour /bin/sh
export DB_PATH=/config/easypour.db
cd /var/app/database/sqlite
sql-migrate up
```

## Required migration

Compile-time constant: `config.RequiredMigration` = `0.base.sql` (menu_items, orders, settings, cvars, and order indexes).

## Existing databases

If `easypour.db` already has the schema from older EasyPour versions (in-process `CREATE TABLE`), the baseline migration uses `IF NOT EXISTS` so the first `sql-migrate up` can stamp the database without failing. After that, use only numbered sql-migrate files for schema changes.
