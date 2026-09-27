package event

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type memEvents struct{ items []Event }

func (m memEvents) List(context.Context) ([]Event, error) { return m.items, nil }
func (m memEvents) ListByMeter(_ context.Context, id string) ([]Event, error) {
	var out []Event
	for _, e := range m.items {
		if e.MeterID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

func TestHandlerList(t *testing.T) {
	ts := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewHandler(memEvents{items: []Event{
		{ID: 1, MeterID: "M-104", Timestamp: ts, Type: TypeOperationalChange},
		{ID: 2, MeterID: "M-106", Timestamp: ts, Type: TypeScheduledOutage},
	}}).Register(mux)

	cases := map[string]int{"/api/events": 2, "/api/events?meter_id=M-106": 1, "/api/events?meter_id=M-000": 0}
	for target, want := range cases {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		var body struct {
			Items []Event `json:"items"`
			Total int     `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("%s: unexpected response %d %s", target, rec.Code, rec.Body.String())
		}
		if body.Total != want || len(body.Items) != want {
			t.Fatalf("%s: got %d events, want %d", target, body.Total, want)
		}
	}
}
