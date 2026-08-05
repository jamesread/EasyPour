# Configuration

EasyPour loads a YAML config file for bootstrap (auth, webhooks, OAuth). Runtime knobs live in SQLite as **configuration variables (cvars)**, edited under **Profile → Settings**.

## Config file location

Search order:

1. `--configdir <dir>/config.yaml` (if the flag is set)
2. `./config.yaml`
3. `./config/config.yaml`
4. `$EASYPOUR_CONFIG_FILE` (if set and the path exists)

Listen address comes from `$PORT` (default `:9654`).

## YAML / env (bootstrap)

| Key | Role |
|-----|------|
| `auth` | httpauthshim session and providers |
| `webhooks` | HTTP webhook URLs for new orders |
| `oauthProviders` | OAuth2 login providers for the login form |

Feature flags are **not** environment variables and are **not** set under a YAML `features:` block.

## Database

Config and a single SQLite file (`easypour.db`) live next to `config.yaml` (typically `/config` in Docker). Menu items, orders, and cvars share that database.

Schema is applied with **sql-migrate** (see [Database migrations](../installation/migrations.md)), not by the Go process. On container start the entrypoint runs `sql-migrate up`; in development use `make migrate` with `DB_PATH` set. The service checks `config.RequiredMigration` against the `migrations` table and exits if it is missing.

On startup, if older `menu.db`, `orders.db`, or `settings.db` files are present, their rows are copied into `easypour.db` with `INSERT OR IGNORE` (admin-chosen values already in the unified DB are kept). The legacy files are not deleted.

`easypour.db` also keeps a legacy `settings` key/value table used only to migrate `apprise_url` into the `apprise_url` cvar once.

## Settings (configuration variables)

Admins manage cvars at **Profile → Settings** (`/admin/settings`). Missing defaults are inserted on startup if they do not already exist. Title, description, category, and ordinal metadata for known cvars are refreshed from application defaults on every startup (including upgrades). Values chosen by admins are **never** overwritten by startup upsert.

The Settings page groups by category and orders by ordinal. Saving a category reloads Init so the header title and other Init fields refresh without a process restart.

| Key | Type | Default | Effect |
|-----|------|---------|--------|
| `site_title` | string | `EasyPour` | Header and browser tab title |
| `apprise_url` | string | empty | Apprise API notify URL for new orders; empty disables notifications |

Existing `apprise_url` values stored in the legacy `settings` table are copied into the cvar on first startup when the cvar is still empty. The Apprise admin page (`/admin/apprise`) remains available to save the URL and send a test notification; it reads and writes the same `apprise_url` cvar.
