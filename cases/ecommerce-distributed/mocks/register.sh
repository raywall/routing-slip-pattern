#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-https://mock.raysouz.studio}"
SERIAL="${SERIAL:-b7af3a9e-6d1a-4b15-9837-3e0f0b47e5b4}"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

register_mock() {
  local id="$1"
  local method="$2"
  local endpoint="$3"
  local status="$4"
  local body_file="$5"
  local response_body

  response_body="$(python3 -c 'import json,sys; print(json.dumps(json.dumps(json.load(open(sys.argv[1])), separators=(",", ":"))))' "${body_file}")"

  printf "Registering ecommerce mock %-38s %s %s\n" "${id}" "${method}" "${endpoint}"
  curl -fsS -X POST "${BASE_URL}/_mock/mocks" \
    -H "Content-Type: application/json" \
    -H "X-Serial-Number: ${SERIAL}" \
    -d @- >/dev/null <<JSON
{
  "id": "${id}",
  "method": "${method}",
  "endpoint": "${endpoint}",
  "responseStatus": ${status},
  "responseBody": ${response_body},
  "responseHeaders": {
    "Content-Type": "application/json"
  }
}
JSON
}

register_mock "ecommerce-order-ord-1001" "GET" "/ecommerce/v1/orders/ORD-1001" "200" "${DIR}/responses/order-ORD-1001.json"
register_mock "ecommerce-customer-cus-1001" "GET" "/ecommerce/v1/customers/CUS-1001" "200" "${DIR}/responses/customer-CUS-1001.json"
register_mock "ecommerce-inventory-ord-1001" "GET" "/ecommerce/v1/inventory/orders/ORD-1001" "200" "${DIR}/responses/inventory-ORD-1001.json"
register_mock "ecommerce-delivery-policy-sp" "GET" "/ecommerce/v1/delivery/policies/SP" "200" "${DIR}/responses/delivery-policy-SP.json"
register_mock "ecommerce-inventory-reservations" "POST" "/ecommerce/v1/inventory/reservations" "200" "${DIR}/responses/inventory-reservation.json"
register_mock "ecommerce-delivery-promises" "POST" "/ecommerce/v1/delivery/promises" "200" "${DIR}/responses/delivery-promise.json"
register_mock "ecommerce-carrier-selection" "POST" "/ecommerce/v1/logistics/carriers/selection" "200" "${DIR}/responses/carrier-selection.json"
register_mock "ecommerce-operational-documents" "POST" "/ecommerce/v1/operations/documents" "200" "${DIR}/responses/operational-document.json"
register_mock "ecommerce-warehouse-picking" "POST" "/ecommerce/v1/warehouse/picking" "200" "${DIR}/responses/warehouse-picking.json"
register_mock "ecommerce-notifications" "POST" "/ecommerce/v1/notifications" "200" "${DIR}/responses/notification.json"
register_mock "ecommerce-order-status-ord-1001" "PATCH" "/ecommerce/v1/orders/ORD-1001/status" "200" "${DIR}/responses/order-status-ORD-1001.json"
register_mock "ecommerce-order-ready-event" "POST" "/ecommerce/v1/events/order-ready" "200" "${DIR}/responses/order-ready-event.json"

printf "Ecommerce mocks registered at %s for serial %s\n" "${BASE_URL}" "${SERIAL}"
