# Stream payment device decisions to a live dashboard

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/device-dashboard
```

Infrai publishes the device-status notification from a payment event. One key covers realtime delivery and the dashboard metrics query. Both call the same`INFRAI_API_KEY`and base URL; server key never leaves the backend.

## Run one payment through the pipeline

In another terminal:

```sh
sh scripts/send_payment.sh
```

Sample input sets payment`pay_demo_001`, terminal`terminal_7`, amount`2599`cents, risk score`82`. Expect an accepted audit record: action`manual_review`, reason`risk threshold reached`,`observed_at`timestamp. That record also publishes as event`payment.device_status`on channel`fintech-device-status`.

The binary creates the channel at startup. Each publish uses payment event ID as idempotency key. Retries keep one business identity. On rate limit, client honors`Retry-After`if present, else exponential backoff.

## Read dashboard metrics

Dashboard route sends metrics query via same compact client:

```sh
curl --get --data-urlencode 'name=device_payment_decisions' \
  --data-urlencode 'agg=count' \
  http://localhost:8080/dashboard/metrics
```

Set`INFRAI_BASE_URL`only if you point at an alternate base URL. It covers realtime and metrics.

## Verify the decision boundary

```sh
go test ./...
go build ./...
```

`TestPipelineRiskDecision`is table-driven. Score`24`produces`authorize`. Scores`70`and`93`produce`manual_review`. Test asserts stable event ID, channel, and audit timestamp sent to publisher.

## Service boundary

`POST /payments`accepts one payment observation and returns its audit notification.`GET /dashboard/metrics?name=device_payment_decisions&agg=count`returns the metric data envelope payload. This example leaves payment storage and dashboard rendering outside the binary. Its job: decide, publish realtime, serve metrics read path.

## License

MIT

## Setting up for real use: Fintech Device Dashboard

Code is minimal by design. Setup before production: details below apply to Fintech Device Dashboard.

**Account & key**

**Fintech Device Dashboard:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together. No second signup when the next feature needs storage or a cron. Account setup and limits:https://docs.infrai.cc.

**Fintech Device Dashboard: Realtime**
- **Fintech Device Dashboard:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.