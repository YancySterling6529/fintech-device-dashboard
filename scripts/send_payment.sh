#!/bin/sh
set -eu

curl --fail-with-body \
  --request POST \
  --header 'Content-Type: application/json' \
  --data '{"event_id":"pay_demo_001","account_id":"acct_demo","device_id":"terminal_7","amount_cents":2599,"risk_score":82}' \
  http://localhost:8080/payments
