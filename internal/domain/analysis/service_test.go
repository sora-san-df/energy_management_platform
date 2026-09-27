package analysis

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

type memRuns struct {
	mu   sync.Mutex
	runs map[int64]Run
	next int64
}

func (m *memRuns) Create(_ context.Context, r *Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	r.ID = m.next
	m.runs[r.ID] = *r
	return nil
}

func (m *memRuns) Update(_ context.Context, r Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[r.ID] = r
	return nil
}

func (m *memRuns) Get(_ context.Context, id int64) (Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return Run{}, ErrRunNotFound
	}
	return r, nil
}

func (m *memRuns) Latest(_ context.Context) (Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[m.next]
	if !ok {
		return Run{}, ErrRunNotFound
	}
	return r, nil
}

type memMeters struct {
	mu       sync.Mutex
	meters   []meter.Meter
	statuses map[string]meter.Status
}

func (m *memMeters) List(context.Context) ([]meter.Meter, error) { return m.meters, nil }
func (m *memMeters) UpdateStatus(_ context.Context, id string, s meter.Status) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[id] = s
	return nil
}

type memReadings struct {
	items []reading.Reading
	err   error
}

func (m *memReadings) ListAll(context.Context) ([]reading.Reading, error) { return m.items, m.err }

type memEvents struct{ items []event.Event }

func (m *memEvents) List(context.Context) ([]event.Event, error) { return m.items, nil }

type memAnomalies struct {
	mu       sync.Mutex
	replaced []string
	stored   map[string][]anomaly.Anomaly
}

func (m *memAnomalies) ReplaceForMeters(_ context.Context, ids []string, items []anomaly.Anomaly) ([]anomaly.Anomaly, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replaced = append(m.replaced, ids...)
	for _, id := range ids {
		delete(m.stored, id)
	}
	out := make([]anomaly.Anomaly, len(items))
	for i, a := range items {
		a.ID = int64(i + 1)
		out[i] = a
		m.stored[a.MeterID] = append(m.stored[a.MeterID], a)
	}
	return out, nil
}

type funcExplainer struct {
	name string
	fn   func(ctx context.Context, req ExplainRequest) (Explanation, error)
}

func (f funcExplainer) Name() string { return f.name }
func (f funcExplainer) Explain(ctx context.Context, req ExplainRequest) (Explanation, error) {
	return f.fn(ctx, req)
}

type fixture struct {
	runs      *memRuns
	meters    *memMeters
	readings  *memReadings
	anomalies *memAnomalies
	deps      Dependencies
}

func newFixture() *fixture {
	var rs []reading.Reading
	rs = append(rs, baseSeries("M-101", 14, 730)...)
	real := baseSeries("M-109", 14, 1050)
	real = real.scale(at(11, 14), end(real), 2.1, 0.74)
	rs = append(rs, real...)
	quality := baseSeries("M-112", 14, 660).corruptElectrical(at(12, 0))
	rs = append(rs, quality...)

	f := &fixture{
		runs: &memRuns{runs: map[int64]Run{}},
		meters: &memMeters{
			meters: []meter.Meter{
				{MeterID: "M-101", Name: "Oficinas", Location: "Sede Central"},
				{MeterID: "M-109", Name: "Subestación Auxiliar", Location: "Planta Norte"},
				{MeterID: "M-112", Name: "Tablero General", Location: "Centro Logístico"},
			},
			statuses: map[string]meter.Status{},
		},
		readings:  &memReadings{items: rs},
		anomalies: &memAnomalies{stored: map[string][]anomaly.Anomaly{}},
	}
	f.deps = Dependencies{
		Runs:      f.runs,
		Meters:    f.meters,
		Readings:  f.readings,
		Events:    &memEvents{items: []event.Event{{ID: 3, MeterID: "M-109", Timestamp: at(11, 14), Type: event.TypeUnknown, Description: "No operational event reported"}}},
		Anomalies: f.anomalies,
		Detector:  NewDetector(DefaultConfig()),
		Clock:     func() time.Time { return time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC) },
	}
	return f
}

func geminiOK() funcExplainer {
	return funcExplainer{name: "gemini-test", fn: func(_ context.Context, req ExplainRequest) (Explanation, error) {
		return Explanation{
			Reason:            "IA: " + req.Finding.MeterID,
			RecommendedAction: "IA acción " + req.Finding.MeterID,
			Confidence:        req.Finding.Confidence,
			Insights:          []string{"insight " + req.Finding.MeterID},
		}, nil
	}}
}

func TestAnalyzeWithGeminiExplanations(t *testing.T) {
	f := newFixture()
	f.deps.Explainer = geminiOK()
	svc := NewService(context.Background(), f.deps)

	run, err := svc.Analyze(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if run.Status != RunStatusCompleted || run.AIStatus != AIStatusOK || run.AIProvider != "gemini-test" {
		t.Fatalf("unexpected run %+v", run)
	}
	if run.MetersAnalyzed != 3 || run.AnomaliesDetected != 2 || run.HighPriority != 2 {
		t.Fatalf("unexpected counters %+v", run)
	}
	for _, s := range run.Steps {
		if s.Status != StepDone || s.Detail == "" {
			t.Fatalf("step %s not completed with detail: %+v", s.Key, s)
		}
	}
	a := f.anomalies.stored["M-109"][0]
	if a.Source != anomaly.SourceGemini || a.Reason != "IA: M-109" || a.RecommendedAction != "IA acción M-109" || len(a.AIInsights) != 1 {
		t.Fatalf("gemini explanation not applied: %+v", a)
	}
	if a.AnalysisID != run.ID {
		t.Fatalf("anomaly must reference run %d, got %d", run.ID, a.AnalysisID)
	}
	if f.meters.statuses["M-109"] != meter.StatusCritical || f.meters.statuses["M-112"] != meter.StatusAlert || f.meters.statuses["M-101"] != meter.StatusOK {
		t.Fatalf("unexpected meter statuses %+v", f.meters.statuses)
	}
	stored, _ := f.runs.Get(context.Background(), run.ID)
	if stored.Status != RunStatusCompleted {
		t.Fatalf("run not persisted as completed: %+v", stored)
	}
}

func TestAnalyzeFallsBackToRulesWhenExplainerFails(t *testing.T) {
	f := newFixture()
	f.deps.Explainer = funcExplainer{name: "gemini-test", fn: func(context.Context, ExplainRequest) (Explanation, error) {
		return Explanation{}, errors.Join(ErrExplainerRateLimited, errors.New("429"))
	}}
	svc := NewService(context.Background(), f.deps)

	run, err := svc.Analyze(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if run.Status != RunStatusCompleted || run.AIStatus != AIStatusFallback {
		t.Fatalf("expected completed run with fallback, got %+v", run)
	}
	a := f.anomalies.stored["M-109"][0]
	if a.Source != anomaly.SourceRules || a.AIError == "" || !strings.Contains(a.Reason, "baseline") {
		t.Fatalf("expected rule based explanation with ai error, got %+v", a)
	}
	for _, s := range run.Steps {
		if s.Key == StepExplanation && !strings.Contains(strings.ToLower(s.Detail), "fallback") {
			t.Fatalf("explanation step must mention fallback: %q", s.Detail)
		}
	}
}

func TestAnalyzeWithoutExplainerIsDisabled(t *testing.T) {
	f := newFixture()
	svc := NewService(context.Background(), f.deps)

	run, err := svc.Analyze(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if run.AIStatus != AIStatusDisabled || run.AIProvider != "rules" {
		t.Fatalf("unexpected AI status %s provider %s", run.AIStatus, run.AIProvider)
	}
	if f.anomalies.stored["M-109"][0].Source != anomaly.SourceRules {
		t.Fatal("anomalies must be explained by rules when AI is disabled")
	}
}

func TestAnalyzePartialAIStatus(t *testing.T) {
	f := newFixture()
	f.deps.Explainer = funcExplainer{name: "gemini-test", fn: func(ctx context.Context, req ExplainRequest) (Explanation, error) {
		if req.Finding.MeterID == "M-112" {
			return Explanation{}, ErrExplainerInvalidResponse
		}
		return geminiOK().fn(ctx, req)
	}}
	svc := NewService(context.Background(), f.deps)

	run, _ := svc.Analyze(context.Background(), nil)
	if run.AIStatus != AIStatusPartial {
		t.Fatalf("expected partial, got %s", run.AIStatus)
	}
	if f.anomalies.stored["M-109"][0].Source != anomaly.SourceGemini || f.anomalies.stored["M-112"][0].Source != anomaly.SourceRules {
		t.Fatal("each anomaly must keep its own source")
	}
}

func TestAnalyzeClampsAIConfidence(t *testing.T) {
	f := newFixture()
	f.deps.Explainer = funcExplainer{name: "gemini-test", fn: func(_ context.Context, req ExplainRequest) (Explanation, error) {
		return Explanation{Reason: "r", RecommendedAction: "a", Confidence: 0.2}, nil
	}}
	svc := NewService(context.Background(), f.deps)

	if _, err := svc.Analyze(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	a := f.anomalies.stored["M-109"][0]
	if a.Confidence < 0.8 {
		t.Fatalf("AI must not move confidence more than the allowed shift, got %.2f", a.Confidence)
	}
}

func TestAnalyzeScopedToMeters(t *testing.T) {
	f := newFixture()
	svc := NewService(context.Background(), f.deps)

	run, err := svc.Analyze(context.Background(), []string{"M-109"})
	if err != nil {
		t.Fatal(err)
	}
	if run.MetersAnalyzed != 1 || run.AnomaliesDetected != 1 {
		t.Fatalf("unexpected scoped run %+v", run)
	}
	if len(f.anomalies.replaced) != 1 || f.anomalies.replaced[0] != "M-109" {
		t.Fatalf("only M-109 anomalies must be replaced, got %v", f.anomalies.replaced)
	}
	if _, touched := f.meters.statuses["M-101"]; touched {
		t.Fatal("meters outside the scope must not be updated")
	}
}

func TestAnalyzeUnknownMeter(t *testing.T) {
	svc := NewService(context.Background(), newFixture().deps)
	if _, err := svc.Analyze(context.Background(), []string{"M-999"}); !errors.Is(err, ErrUnknownMeter) {
		t.Fatalf("expected ErrUnknownMeter, got %v", err)
	}
}

func TestAnalyzeFailsWhenReadingsUnavailable(t *testing.T) {
	f := newFixture()
	f.readings.err = errors.New("disk on fire")
	svc := NewService(context.Background(), f.deps)

	run, err := svc.Analyze(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if run.Status != RunStatusFailed || !strings.Contains(run.Error, "disk on fire") {
		t.Fatalf("expected failed run, got %+v", run)
	}
	stored, _ := f.runs.Get(context.Background(), run.ID)
	if stored.Status != RunStatusFailed {
		t.Fatal("failed run must be persisted")
	}
}

func TestStartRunsInBackground(t *testing.T) {
	f := newFixture()
	svc := NewService(context.Background(), f.deps)

	run, err := svc.Start(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == 0 || run.Status != RunStatusRunning {
		t.Fatalf("unexpected started run %+v", run)
	}
	svc.Wait()

	got, err := svc.Get(context.Background(), run.ID)
	if err != nil || got.Status != RunStatusCompleted {
		t.Fatalf("expected completed run, got %+v (%v)", got, err)
	}
	latest, err := svc.Latest(context.Background())
	if err != nil || latest.ID != run.ID {
		t.Fatalf("unexpected latest %+v (%v)", latest, err)
	}
}

func TestStartRejectsConcurrentRuns(t *testing.T) {
	f := newFixture()
	release := make(chan struct{})
	f.deps.Explainer = funcExplainer{name: "slow", fn: func(ctx context.Context, req ExplainRequest) (Explanation, error) {
		<-release
		return geminiOK().fn(ctx, req)
	}}
	svc := NewService(context.Background(), f.deps)

	first, err := svc.Start(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	current, err := svc.Start(context.Background(), nil)
	if !errors.Is(err, ErrAnalysisInProgress) || current.ID != first.ID {
		t.Fatalf("expected ErrAnalysisInProgress with running run %d, got %d (%v)", first.ID, current.ID, err)
	}
	close(release)
	svc.Wait()

	if _, err := svc.Start(context.Background(), nil); err != nil {
		t.Fatalf("a new run must be allowed after completion: %v", err)
	}
	svc.Wait()
}

func TestStartValidatesScopeSynchronously(t *testing.T) {
	svc := NewService(context.Background(), newFixture().deps)
	if _, err := svc.Start(context.Background(), []string{"M-999"}); !errors.Is(err, ErrUnknownMeter) {
		t.Fatalf("expected ErrUnknownMeter, got %v", err)
	}
}

func TestDescribeAIError(t *testing.T) {
	cases := map[error]string{
		ErrExplainerRateLimited:     "límite",
		ErrExplainerTimeout:         "tiempo",
		ErrExplainerInvalidResponse: "respuesta",
		ErrExplainerUnavailable:     "disponible",
		errors.New("x"):             "Gemini",
	}
	for err, fragment := range cases {
		if got := describeAIError(err); !strings.Contains(got, fragment) {
			t.Errorf("describeAIError(%v) = %q, want it to contain %q", err, got, fragment)
		}
	}
}
