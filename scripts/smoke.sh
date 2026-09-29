#!/usr/bin/env bash
set -euo pipefail

base_url="${BASE_URL:-http://localhost:8080}"
mailpit_url="${MAILPIT_URL:-http://localhost:8025}"
owner_email="smoke-$(date +%s)-$$@example.com"

request() {
  local method="$1" path="$2" body="${3:-}"
  if [[ -n "$body" ]]; then
    curl --fail --silent --show-error -X "$method" "${base_url}${path}" \
      -H 'Content-Type: application/json' -H "Authorization: Bearer ${access_token:-}" --data "$body"
  else
    curl --fail --silent --show-error -X "$method" "${base_url}${path}" \
      -H "Authorization: Bearer ${access_token:-}"
  fi
}

request GET /healthz | jq -e '.status == "ok"' >/dev/null
request GET /readyz | jq -e '.status == "ok"' >/dev/null
request GET / | grep -q 'StockPilot'
request GET /reset-password | grep -q 'StockPilot'
request GET /app.js | grep -q 'refreshToken'

organization="$(request POST /organizations "$(jq -nc --arg email "$owner_email" '{name:"Smoke Workspace",owner_email:$email,owner_name:"Owner",owner_password:"password123"}')")"
organization_id="$(jq -er '.id' <<<"$organization")"
prefix="/organizations/${organization_id}"
tokens="$(request POST "${prefix}/sessions" "$(jq -nc --arg email "$owner_email" '{email:$email,password:"password123"}')")"
access_token="$(jq -er '.access_token' <<<"$tokens")"

warehouse_id="$(request POST "${prefix}/warehouses" '{"name":"Main warehouse"}' | jq -er '.id')"
product_id="$(request POST "${prefix}/products" '{"name":"Demo widget"}' | jq -er '.id')"
sku_id="$(request POST "${prefix}/products/${product_id}/skus" '{"code":"WIDGET-01","name":"Widget","price_cents":1299}' | jq -er '.id')"
stock_path="${prefix}/warehouses/${warehouse_id}/skus/${sku_id}/stock"
request POST "${stock_path}/receive" '{"quantity":10,"idempotency_key":"smoke-receive-1"}' | jq -e '.on_hand == 10 and .available == 10' >/dev/null

order_body="$(jq -nc --arg warehouse "$warehouse_id" --arg sku "$sku_id" '{idempotency_key:"smoke-order-1",lines:[{warehouse_id:$warehouse,sku_id:$sku,quantity:2}]}')"
order_id="$(request POST "${prefix}/orders" "$order_body" | jq -er 'select(.status == "pending" and .total_cents == 2598) | .id')"
request GET "$stock_path" | jq -e '.on_hand == 10 and .reserved == 2 and .available == 8' >/dev/null
request POST "${prefix}/orders/${order_id}/confirm" | jq -e '.status == "confirmed"' >/dev/null
request POST "${prefix}/orders/${order_id}/pay" | jq -e '.status == "paid"' >/dev/null
request POST "${prefix}/orders/${order_id}/ship" | jq -e '.status == "shipped"' >/dev/null
request POST "${prefix}/orders/${order_id}/complete" | jq -e '.status == "completed"' >/dev/null
request GET "$stock_path" | jq -e '.on_hand == 8 and .reserved == 0 and .available == 8' >/dev/null
request GET "${prefix}/reports/summary?low_stock_threshold=8" | jq -e '.orders.completed == 1 and .orders.confirmed_cents == 2598 and .inventory.low_stock == 1' >/dev/null
request GET "${prefix}/exports/stocks/csv" | grep -q "$sku_id"

request PUT "${prefix}/alerts/stock" '{"threshold":8,"enabled":true}' | jq -e '.threshold == 8 and .enabled == true' >/dev/null
request POST "${prefix}/sessions/password-reset" "$(jq -nc --arg email "$owner_email" '{email:$email}')" | jq -e '.message != ""' >/dev/null

delivered=false
for _ in {1..30}; do
  if curl --fail --silent --show-error "${mailpit_url}/api/v1/messages" | jq -e --arg email "$owner_email" 'any(.messages[]?; .Subject == "Reset your workspace password" and any(.To[]?; .Address == $email))' >/dev/null; then
    delivered=true
    break
  fi
  sleep 1
done
if [[ "$delivered" != true ]]; then
  echo "password reset email not delivered to Mailpit" >&2
  exit 1
fi

echo "compose smoke test passed: organization, inventory, orders, reports, CSV, and email"
