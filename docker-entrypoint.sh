#!/bin/sh
set -e

DRIVER="${DB_DRIVER:-sqlite}"
case "$DRIVER" in
  mysql|pgsql|sqlite) ;;
  *)
    echo "unsupported DB_DRIVER: $DRIVER" >&2
    exit 1
    ;;
esac

CONFIG_DIR="/config"
DEFAULTS="/usr/share/easypour/defaults"
if [ -n "${EASYPOUR_CONFIG_FILE:-}" ]; then
  CONFIG_DIR="$(dirname "$EASYPOUR_CONFIG_FILE")"
fi

if [ "$DRIVER" = "sqlite" ]; then
  mkdir -p "$CONFIG_DIR" "$CONFIG_DIR/images"
  for f in config.yaml menu.yaml; do
    if [ ! -f "$CONFIG_DIR/$f" ] && [ -f "$DEFAULTS/$f" ]; then
      cp "$DEFAULTS/$f" "$CONFIG_DIR/$f"
    fi
  done
  if [ -z "${DB_PATH:-}" ]; then
    if [ -n "${EASYPOUR_CONFIG_FILE:-}" ]; then
      export DB_PATH="$(dirname "$EASYPOUR_CONFIG_FILE")/easypour.db"
    else
      export DB_PATH="/config/easypour.db"
    fi
  fi
fi

cd "/var/app/database/${DRIVER}" && sql-migrate up
exec /usr/bin/easypour-service "$@"
