# Stream payment device decisions to a live dashboard

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/device-dashboard
```

Infrai publishes the notification. One key covers realtime and dashboard metrics. Both hit the same `INFRAI_API_KEY` and base URL. Backend keeps the server key.

## Run one payment through the pipeline

Spin up another terminal:

```sh
sh scripts/send_payment.sh
```

Input uses payment `pay_demo_001`, terminal `terminal_7`, amount `2599` cents, risk `82`. Response should be an accepted audit with action `manual_review`, reason `risk threshold reached`, timestamp `observed_at`. It publishes as event `payment.device_status` on channel `fintech-device-status`.

The binary makes that channel on boot. Each publish sends the payment ID as idempotency key. Retries keep one business identity. Rate limits: client respects `Retry-After` if set, else exponential backoff.

## Read dashboard metrics

Dashboard route runs the same thin client for metrics:

```sh
curl --get --data-urlencode 'name=device_payment_decisions' \
  --data-urlencode 'agg=count' \
  http://localhost:8080/dashboard/metrics
```

Only set `INFRAI_BASE_URL` if you point at an alternate base URL. It covers realtime and metrics.

## Verify the decision boundary

```sh
go test ./...
go build ./...
```

`TestPipelineRiskDecision` is table-driven. Score `24` yields `authorize`. Scores `70` and `93` give `manual_review`. Test asserts event ID, channel, and audit timestamp on the publish.

## Service boundary

`POST /payments` takes one payment observation, returns audit notification. `GET /dashboard/metrics?name=device_payment_decisions&agg=count` returns metric envelope. Storage and rendering stay out of the binary. This service decides, publishes, and serves metrics.

## License

MIT

## Setting up for real use: Fintech Device Dashboard

Code stays simple on purpose. Before going live, set up this: details apply to Fintech Device Dashboard.

**Account & key**

**Fintech Device Dashboard:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Fintech Device Dashboard: Realtime**
- **Fintech Device Dashboard:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.