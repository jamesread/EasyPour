-- +migrate Up
-- IF NOT EXISTS: safe when stamping a DB that already had schema from store.Init().
CREATE TABLE IF NOT EXISTS menu_items (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	supports_sugar INTEGER NOT NULL,
	supports_milk INTEGER NOT NULL,
	image_url TEXT NOT NULL DEFAULT '',
	category TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS orders (
	id TEXT PRIMARY KEY,
	menu_item_id TEXT NOT NULL,
	username TEXT NOT NULL,
	add_sugar INTEGER NOT NULL,
	add_milk INTEGER NOT NULL,
	sugar_amount INTEGER NOT NULL,
	milk_amount INTEGER NOT NULL,
	status TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	group_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_orders_username ON orders(username);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_group_id ON orders(group_id);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cvars (
	cvar_key TEXT NOT NULL PRIMARY KEY,
	cvar_value_int INTEGER NULL,
	cvar_value_string TEXT NULL,
	cvar_main_type TEXT NOT NULL,
	cvar_title TEXT NOT NULL DEFAULT '',
	cvar_description TEXT NOT NULL DEFAULT '',
	cvar_category TEXT NOT NULL DEFAULT '',
	cvar_ordinal INTEGER NOT NULL DEFAULT 0
);

-- +migrate Down
DROP INDEX IF EXISTS idx_orders_group_id;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_username;
DROP TABLE IF EXISTS cvars;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS menu_items;
