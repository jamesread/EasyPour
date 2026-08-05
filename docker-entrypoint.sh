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

if [ "$DRIVER" = "sqlite" ]; then
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
