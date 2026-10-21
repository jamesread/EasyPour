#!/usr/bin/env bash
# Seeds the throwaway backend with fictional orders and settings via its public
# API (Connect JSON). Usage: seed.sh <backend-base-url>
set -euo pipefail

base="${1:?backend base url}"
here="$(cd "$(dirname "$0")" && pwd)"
jars="$here/tmp/cookies"
api="$base/easypour.v1.EasyPourService"
mkdir -p "$jars"

login() {
	curl -fsS -c "$jars/$1" -H 'Content-Type: application/json' \
		-d "{\"username\": \"$1\", \"password\": \"demo-password\"}" "$base/login" >/dev/null
}

# rpc <user> <method> <json>
rpc() {
	curl -fsS -b "$jars/$1" -X POST -H 'Content-Type: application/json' -d "$3" "$api/$2"
}

for user in alex sam jordan; do
	login "$user"
done

# Orders are placed by their owner; status changes are made by the admin (alex).
jq -c '.[]' "$here/seed/orders.json" | while read -r basket; do
	user="$(jq -r .user <<<"$basket")"
	status="$(jq -r .status <<<"$basket")"
	group=""
	while read -r item; do
		req="$(jq -c --arg g "$group" '. + {groupId: $g}' <<<"$item")"
		id="$(rpc "$user" OrderDrink "$req" | jq -r .orderId)"
		group="${group:-$id}"
		if [ "$status" = "preparing" ]; then
			rpc alex UpdateOrderStatus "{\"orderId\": \"$id\", \"status\": \"preparing\"}" >/dev/null
		elif [ "$status" = "delivered" ]; then
			rpc alex UpdateOrderStatus "{\"orderId\": \"$id\", \"status\": \"delivered\"}" >/dev/null
		fi
	done < <(jq -c '.items[]' <<<"$basket")
	sleep 1
done

# Settings last, so the Apprise URL is never contacted while seeding orders.
jq -c 'to_entries[] | {key, valueString: .value}' "$here/seed/config.json" | while read -r kv; do
	rpc alex UpdateCvar "$kv" >/dev/null
done
