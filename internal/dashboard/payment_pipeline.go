package dashboard

import (
	"context"
	"errors"
	"fmt"
	"time"

	api "github.com/infrai-examples/fintech-device-dashboard/internal/infrai"
)

const DeviceChannel = "fintech-device-status"

type PaymentEvent struct {
	EventID     string `json:"event_id"`
	AccountID   string `json:"account_id"`
	DeviceID    string `json:"device_id"`
	AmountCents int64  `json:"amount_cents"`
	RiskScore   int    `json:"risk_score"`
}

type AuditNotification struct {
	EventID     string    `json:"event_id"`
	DeviceID    string    `json:"device_id"`
	AmountCents int64     `json:"amount_cents"`
	RiskScore   int       `json:"risk_score"`
	Action      string    `json:"action"`
	Reason      string    `json:"reason"`
	ObservedAt  time.Time `json:"observed_at"`
}

type Publisher interface {
	Publish(context.Context, api.DeviceEvent) error
}

type Pipeline struct {
	publisher Publisher
	now       func() time.Time
}

func NewPipeline(publisher Publisher) *Pipeline {
	return &Pipeline{publisher: publisher, now: time.Now}
}

func (p *Pipeline) Process(ctx context.Context, event PaymentEvent) (AuditNotification, error) {
	if event.EventID == "" || event.AccountID == "" || event.DeviceID == "" {
		return AuditNotification{}, errors.New("event_id, account_id, and device_id are required")
	}
	if event.AmountCents <= 0 || event.RiskScore < 0 || event.RiskScore > 100 {
		return AuditNotification{}, errors.New("amount_cents must be positive and risk_score must be between 0 and 100")
	}

	action, reason := DecideAction(event.RiskScore)
	notification := AuditNotification{
		EventID: event.EventID, DeviceID: event.DeviceID, AmountCents: event.AmountCents,
		RiskScore: event.RiskScore, Action: action, Reason: reason, ObservedAt: p.now().UTC(),
	}
	err := p.publisher.Publish(ctx, api.DeviceEvent{
		Channel: DeviceChannel, Event: "payment.device_status", Data: notification,
		AccountID: event.AccountID, EventID: event.EventID,
	})
	if err != nil {
		return AuditNotification{}, fmt.Errorf("publish device status: %w", err)
	}
	return notification, nil
}

func DecideAction(riskScore int) (action, reason string) {
	if riskScore >= 70 {
		return "manual_review", "risk threshold reached"
	}
	return "authorize", "risk below review threshold"
}
