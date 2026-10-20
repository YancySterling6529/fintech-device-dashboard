package dashboard

import (
	"context"
	"testing"
	"time"

	api "github.com/infrai-examples/fintech-device-dashboard/internal/infrai"
)

type recordingPublisher struct {
	event api.DeviceEvent
}

func (p *recordingPublisher) Publish(_ context.Context, event api.DeviceEvent) error {
	p.event = event
	return nil
}

func TestPipelineRiskDecision(t *testing.T) {
	tests := []struct {
		name       string
		riskScore  int
		wantAction string
	}{
		{name: "ordinary payment is authorized", riskScore: 24, wantAction: "authorize"},
		{name: "threshold payment is reviewed", riskScore: 70, wantAction: "manual_review"},
		{name: "higher risk payment is reviewed", riskScore: 93, wantAction: "manual_review"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &recordingPublisher{}
			pipeline := NewPipeline(publisher)
			observedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			pipeline.now = func() time.Time { return observedAt }

			got, err := pipeline.Process(context.Background(), PaymentEvent{
				EventID: "pay_01", AccountID: "acct_42", DeviceID: "terminal_7",
				AmountCents: 2599, RiskScore: tt.riskScore,
			})
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if got.Action != tt.wantAction {
				t.Fatalf("Action = %q, want %q", got.Action, tt.wantAction)
			}
			if got.ObservedAt != observedAt {
				t.Fatalf("ObservedAt = %v, want %v", got.ObservedAt, observedAt)
			}
			if publisher.event.EventID != "pay_01" || publisher.event.Channel != DeviceChannel {
				t.Fatalf("published event = %#v", publisher.event)
			}
		})
	}
}
