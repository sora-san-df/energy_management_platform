package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/genai"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/analysis"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
)

type scriptedGenerator struct {
	responses []string
	errs      []error
	calls     int
	system    string
	prompt    string
}

func (s *scriptedGenerator) Generate(ctx context.Context, system, prompt string) (string, error) {
	i := s.calls
	s.calls++
	s.system, s.prompt = system, prompt
	var err error
	if i < len(s.errs) {
		err = s.errs[i]
	}
	if err != nil {
		return "", err
	}
	if i < len(s.responses) {
		return s.responses[i], nil
	}
	return s.responses[len(s.responses)-1], nil
}

const validJSON = `{"reason":"Consumo 109% sobre baseline sin evento.","recommended_action":"Inspeccionar la subestación.","confidence":0.95,"insights":["Corriente duplicada","FP cayó a 0,74"]}`

func request() analysis.ExplainRequest {
	start := time.Date(2026, 9, 12, 14, 0, 0, 0, time.UTC)
	return analysis.ExplainRequest{
		MeterName: "Subestación Auxiliar",
		Location:  "Planta Norte",
		Finding: analysis.Finding{
			MeterID:           "M-109",
			Anomaly:           true,
			Type:              anomaly.TypeRealAnomaly,
			Severity:          anomaly.SeverityHigh,
			Confidence:        0.97,
			Reason:            "Consumo 109,2% por encima del baseline",
			RecommendedAction: "Investigar medidor e instalación",
			BaselineDailyKWh:  1055,
			CurrentDailyKWh:   2208,
			VariationPct:      109.2,
			Shift:             &analysis.Shift{Start: start, Direction: analysis.DirectionUp, Persistent: true},
			RelatedEvent:      &event.Event{MeterID: "M-109", Timestamp: start, Type: event.TypeUnknown, Description: "No operational event reported"},
			Evidence:          []anomaly.Evidence{{Label: "Variación vs baseline", Value: "+109,2%"}},
			AffectedVariables: []string{"consumption_kwh", "power_factor"},
		},
	}
}

func newTestExplainer(gen Generator) (*Explainer, *[]time.Duration) {
	var sleeps []time.Duration
	e := NewExplainer(gen, Options{Model: "gemini-test", Timeout: time.Second, MaxRetries: 2, Backoff: 100 * time.Millisecond})
	e.sleep = func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}
	return e, &sleeps
}

func TestExplainSuccess(t *testing.T) {
	gen := &scriptedGenerator{responses: []string{validJSON}}
	e, _ := newTestExplainer(gen)

	exp, err := e.Explain(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if exp.Reason != "Consumo 109% sobre baseline sin evento." || exp.RecommendedAction != "Inspeccionar la subestación." || exp.Confidence != 0.95 || len(exp.Insights) != 2 {
		t.Fatalf("unexpected explanation %+v", exp)
	}
	if e.Name() != "gemini-test" {
		t.Fatalf("name = %s", e.Name())
	}
	for _, fragment := range []string{"M-109", "REAL_ANOMALY", "HIGH", "Subestación Auxiliar", "+109,2%", "UNKNOWN"} {
		if !strings.Contains(gen.prompt, fragment) {
			t.Errorf("prompt must contain %q", fragment)
		}
	}
	if !strings.Contains(gen.system, "español") || !strings.Contains(gen.system, "JSON") {
		t.Errorf("system prompt must require Spanish JSON output: %s", gen.system)
	}
}

func TestExplainRetriesOnRateLimit(t *testing.T) {
	gen := &scriptedGenerator{
		errs:      []error{genai.APIError{Code: 429, Message: "quota"}, nil},
		responses: []string{"", validJSON},
	}
	e, sleeps := newTestExplainer(gen)

	if _, err := e.Explain(context.Background(), request()); err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if gen.calls != 2 || len(*sleeps) != 1 || (*sleeps)[0] != 100*time.Millisecond {
		t.Fatalf("expected one retry with backoff, got calls=%d sleeps=%v", gen.calls, *sleeps)
	}
}

func TestExplainGivesUpAfterMaxRetries(t *testing.T) {
	rateLimited := genai.APIError{Code: 429}
	gen := &scriptedGenerator{errs: []error{rateLimited, rateLimited, rateLimited, rateLimited}}
	e, sleeps := newTestExplainer(gen)

	_, err := e.Explain(context.Background(), request())
	if !errors.Is(err, analysis.ErrExplainerRateLimited) {
		t.Fatalf("expected ErrExplainerRateLimited, got %v", err)
	}
	if gen.calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", gen.calls)
	}
	if len(*sleeps) != 2 || (*sleeps)[1] != 200*time.Millisecond {
		t.Fatalf("expected exponential backoff, got %v", *sleeps)
	}
}

func TestExplainRetriesServerErrors(t *testing.T) {
	gen := &scriptedGenerator{errs: []error{genai.APIError{Code: 503}, nil}, responses: []string{"", validJSON}}
	e, _ := newTestExplainer(gen)
	if _, err := e.Explain(context.Background(), request()); err != nil {
		t.Fatalf("expected recovery from 503, got %v", err)
	}
}

func TestExplainDoesNotRetryAuthErrors(t *testing.T) {
	gen := &scriptedGenerator{errs: []error{genai.APIError{Code: 401, Message: "bad key"}}}
	e, _ := newTestExplainer(gen)

	_, err := e.Explain(context.Background(), request())
	if !errors.Is(err, analysis.ErrExplainerUnavailable) {
		t.Fatalf("expected ErrExplainerUnavailable, got %v", err)
	}
	if gen.calls != 1 {
		t.Fatalf("auth errors must not be retried, got %d calls", gen.calls)
	}
}

func TestExplainTimeout(t *testing.T) {
	gen := &scriptedGenerator{errs: []error{context.DeadlineExceeded, context.DeadlineExceeded, context.DeadlineExceeded}}
	e, _ := newTestExplainer(gen)

	_, err := e.Explain(context.Background(), request())
	if !errors.Is(err, analysis.ErrExplainerTimeout) {
		t.Fatalf("expected ErrExplainerTimeout, got %v", err)
	}
}

func TestExplainStopsWhenParentContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	gen := &scriptedGenerator{errs: []error{context.Canceled}}
	e, _ := newTestExplainer(gen)

	if _, err := e.Explain(ctx, request()); err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if gen.calls > 1 {
		t.Fatalf("must not retry a cancelled context, got %d calls", gen.calls)
	}
}

func TestExplainInvalidResponse(t *testing.T) {
	for _, body := range []string{"", "no json here", `{"reason":"","recommended_action":"x","confidence":0.9}`, `{"reason":"x","recommended_action":"","confidence":0.9}`} {
		gen := &scriptedGenerator{responses: []string{body}}
		e, _ := newTestExplainer(gen)
		if _, err := e.Explain(context.Background(), request()); !errors.Is(err, analysis.ErrExplainerInvalidResponse) {
			t.Errorf("body %q: expected ErrExplainerInvalidResponse, got %v", body, err)
		}
	}
}

func TestParseExplanationHandlesCodeFences(t *testing.T) {
	exp, err := ParseExplanation("```json\n" + validJSON + "\n```")
	if err != nil || exp.Confidence != 0.95 {
		t.Fatalf("unexpected %+v %v", exp, err)
	}
}

func TestParseExplanationNormalisesConfidence(t *testing.T) {
	exp, err := ParseExplanation(`{"reason":"r","recommended_action":"a","confidence":96}`)
	if err != nil || exp.Confidence != 0.96 {
		t.Fatalf("percentage confidence should be normalised, got %+v %v", exp, err)
	}
	if _, err := ParseExplanation(`{"reason":"r","recommended_action":"a","confidence":-1}`); !errors.Is(err, analysis.ErrExplainerInvalidResponse) {
		t.Fatalf("negative confidence must be invalid, got %v", err)
	}
	exp, err = ParseExplanation(`{"reason":"r","recommended_action":"a"}`)
	if err != nil || exp.Confidence != 0 || exp.Insights == nil {
		t.Fatalf("missing confidence is allowed and insights default to empty, got %+v %v", exp, err)
	}
}

func TestNewGenaiGeneratorRequiresAPIKey(t *testing.T) {
	if _, err := NewGenaiGenerator(context.Background(), "", "gemini-2.5-flash"); !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}
}
