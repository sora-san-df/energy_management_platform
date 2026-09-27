package analysis

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/sora-san-df/ai-energy-management-platform/internal/analytics"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/meter"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/reading"
)

const rulesProvider = "rules"

type Dependencies struct {
	Runs               RunRepository
	Meters             MeterSource
	Readings           ReadingSource
	Events             EventSource
	Anomalies          AnomalyStore
	Explainer          Explainer
	Detector           *Detector
	Clock              func() time.Time
	StepDelay          time.Duration
	ExplainConcurrency int
	ConfidenceShift    float64
	RunTimeout         time.Duration
	Logger             *slog.Logger
}

type Service struct {
	deps     Dependencies
	baseCtx  context.Context
	mu       sync.Mutex
	activeID int64
	wg       sync.WaitGroup
}

func NewService(baseCtx context.Context, deps Dependencies) *Service {
	if deps.Detector == nil {
		deps.Detector = NewDetector(DefaultConfig())
	}
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	if deps.ExplainConcurrency <= 0 {
		deps.ExplainConcurrency = 4
	}
	if deps.ConfidenceShift <= 0 {
		deps.ConfidenceShift = 0.10
	}
	if deps.RunTimeout <= 0 {
		deps.RunTimeout = 3 * time.Minute
	}
	if deps.Logger == nil {
		deps.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Service{deps: deps, baseCtx: baseCtx}
}

func (s *Service) providerName() string {
	if s.deps.Explainer == nil {
		return rulesProvider
	}
	return s.deps.Explainer.Name()
}

func (s *Service) Get(ctx context.Context, id int64) (Run, error) {
	return s.deps.Runs.Get(ctx, id)
}

func (s *Service) Latest(ctx context.Context) (Run, error) {
	return s.deps.Runs.Latest(ctx)
}

func (s *Service) Wait() {
	s.wg.Wait()
}

func (s *Service) resolveScope(ctx context.Context, meterIDs []string) ([]meter.Meter, error) {
	all, err := s.deps.Meters.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(meterIDs) == 0 {
		return all, nil
	}
	index := map[string]meter.Meter{}
	for _, m := range all {
		index[m.MeterID] = m
	}
	var out []meter.Meter
	seen := map[string]bool{}
	for _, id := range meterIDs {
		m, ok := index[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownMeter, id)
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Service) newRun(ctx context.Context, meterIDs []string) (Run, []meter.Meter, error) {
	meters, err := s.resolveScope(ctx, meterIDs)
	if err != nil {
		return Run{}, nil, err
	}
	run := NewRun(meterIDs, s.deps.Clock())
	run.AIProvider = s.providerName()
	if err := s.deps.Runs.Create(ctx, &run); err != nil {
		return Run{}, nil, fmt.Errorf("create analysis run: %w", err)
	}
	return run, meters, nil
}

func (s *Service) Analyze(ctx context.Context, meterIDs []string) (Run, error) {
	run, meters, err := s.newRun(ctx, meterIDs)
	if err != nil {
		return Run{}, err
	}
	err = s.execute(ctx, &run, meters)
	return run, err
}

func (s *Service) Start(ctx context.Context, meterIDs []string) (Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeID != 0 {
		if current, err := s.deps.Runs.Get(ctx, s.activeID); err == nil {
			return current, ErrAnalysisInProgress
		}
	}
	run, meters, err := s.newRun(ctx, meterIDs)
	if err != nil {
		return Run{}, err
	}
	s.activeID = run.ID
	s.wg.Add(1)
	go func(run Run) {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			s.activeID = 0
			s.mu.Unlock()
		}()
		runCtx, cancel := context.WithTimeout(s.baseCtx, s.deps.RunTimeout)
		defer cancel()
		if err := s.execute(runCtx, &run, meters); err != nil {
			s.deps.Logger.Error("analysis run failed", "run_id", run.ID, "error", err)
		}
	}(run)
	return run, nil
}

func (s *Service) persist(run Run) {
	if err := s.deps.Runs.Update(context.WithoutCancel(s.baseCtx), run); err != nil {
		s.deps.Logger.Error("persist analysis run", "run_id", run.ID, "error", err)
	}
}

func (s *Service) begin(ctx context.Context, run *Run, step string) error {
	if s.deps.StepDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.deps.StepDelay):
		}
	}
	run.StartStep(step, s.deps.Clock())
	s.persist(*run)
	return ctx.Err()
}

func (s *Service) complete(run *Run, step, detail string) {
	run.CompleteStep(step, detail, s.deps.Clock())
	s.persist(*run)
}

type meterWork struct {
	meter    meter.Meter
	readings []reading.Reading
	baseline Baseline
	shift    *Shift
	quality  QualityReport
	change   *ElectricalChange
	related  *event.Event
	finding  Finding
}

func (s *Service) execute(ctx context.Context, run *Run, meters []meter.Meter) error {
	fail := func(err error) error {
		run.Fail(err, s.deps.Clock())
		s.persist(*run)
		return err
	}
	d := s.deps.Detector

	if err := s.begin(ctx, run, StepReadings); err != nil {
		return fail(err)
	}
	all, err := s.deps.Readings.ListAll(ctx)
	if err != nil {
		return fail(fmt.Errorf("load readings: %w", err))
	}
	byMeter := map[string][]reading.Reading{}
	for _, r := range all {
		byMeter[r.MeterID] = append(byMeter[r.MeterID], r)
	}
	work := make([]*meterWork, len(meters))
	total := 0
	for i, m := range meters {
		work[i] = &meterWork{meter: m, readings: SortReadings(byMeter[m.MeterID])}
		total += len(work[i].readings)
	}
	s.complete(run, StepReadings, fmt.Sprintf("%s lecturas de %d medidores", formatNumber(float64(total), 0), len(meters)))

	if err := s.begin(ctx, run, StepBaseline); err != nil {
		return fail(err)
	}
	for _, w := range work {
		w.baseline = d.Baseline(w.readings)
	}
	s.complete(run, StepBaseline, "Perfil horario por medidor (mediana del período) y consumo diario de referencia")

	if err := s.begin(ctx, run, StepDetection); err != nil {
		return fail(err)
	}
	shifts, qualityIssues := 0, 0
	for _, w := range work {
		if len(w.readings) == 0 {
			continue
		}
		w.shift = d.DetectShift(w.readings, w.baseline)
		w.quality = d.CheckQuality(w.readings)
		if w.shift != nil {
			shifts++
		}
		if d.hasQualityProblem(w.quality) {
			qualityIssues++
		}
	}
	s.complete(run, StepDetection, fmt.Sprintf("%d cambios de consumo · %d medidores con lecturas inconsistentes", shifts, qualityIssues))

	if err := s.begin(ctx, run, StepCorrelation); err != nil {
		return fail(err)
	}
	electrical := 0
	for _, w := range work {
		w.change = d.Correlate(w.readings, w.shift)
		if w.change != nil && w.change.Significant {
			electrical++
		}
	}
	s.complete(run, StepCorrelation, fmt.Sprintf("%d cambios con alteración eléctrica significativa (corriente, voltaje, FP)", electrical))

	if err := s.begin(ctx, run, StepEvents); err != nil {
		return fail(err)
	}
	events, err := s.deps.Events.List(ctx)
	if err != nil {
		return fail(fmt.Errorf("load events: %w", err))
	}
	related, explained := 0, 0
	var findings []*meterWork
	for _, w := range work {
		if len(w.readings) == 0 {
			continue
		}
		w.related = d.MatchEvent(w.meter.MeterID, events, w.shift, w.quality)
		w.finding = d.Classify(w.meter.MeterID, w.baseline, w.shift, w.change, w.quality, w.related)
		if w.related != nil {
			related++
		}
		if w.finding.Explained {
			explained++
		}
		if w.finding.Anomaly {
			findings = append(findings, w)
		}
	}
	s.complete(run, StepEvents, fmt.Sprintf("%d eventos relacionados · %d explican el cambio observado", related, explained))

	if err := s.begin(ctx, run, StepExplanation); err != nil {
		return fail(err)
	}
	anomalies, aiStatus, detail := s.explain(ctx, run.ID, findings)
	s.complete(run, StepExplanation, detail)

	if err := s.begin(ctx, run, StepRecommendation); err != nil {
		return fail(err)
	}
	ids := make([]string, len(meters))
	for i, m := range meters {
		ids[i] = m.MeterID
	}
	saved, err := s.deps.Anomalies.ReplaceForMeters(ctx, ids, anomaly.Rank(anomalies))
	if err != nil {
		return fail(fmt.Errorf("store anomalies: %w", err))
	}
	grouped := map[string][]anomaly.Anomaly{}
	for _, a := range saved {
		grouped[a.MeterID] = append(grouped[a.MeterID], a)
	}
	for _, id := range ids {
		if err := s.deps.Meters.UpdateStatus(ctx, id, meter.StatusFromAnomalies(grouped[id])); err != nil {
			return fail(fmt.Errorf("update meter %s: %w", id, err))
		}
	}
	high := 0
	for _, a := range saved {
		if a.IsHighPriority() {
			high++
		}
	}
	s.complete(run, StepRecommendation, fmt.Sprintf("%d acciones recomendadas · %d requieren atención prioritaria", len(saved), high))

	run.Complete(saved, len(meters), aiStatus, s.deps.Clock())
	s.persist(*run)
	return nil
}

func (s *Service) explain(ctx context.Context, runID int64, work []*meterWork) ([]anomaly.Anomaly, AIStatus, string) {
	anomalies := make([]anomaly.Anomaly, len(work))
	for i, w := range work {
		anomalies[i] = w.finding.ToAnomaly(runID)
	}
	if s.deps.Explainer == nil {
		return anomalies, AIStatusDisabled, "Explicaciones generadas por el motor de reglas (Gemini no configurado)"
	}
	if len(work) == 0 {
		return anomalies, AIStatusOK, "Sin anomalías que explicar"
	}

	errs := make([]error, len(work))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(s.deps.ExplainConcurrency)
	for i, w := range work {
		g.Go(func() error {
			exp, err := s.deps.Explainer.Explain(gctx, ExplainRequest{Finding: w.finding, MeterName: w.meter.Name, Location: w.meter.Location})
			if err != nil {
				errs[i] = err
				anomalies[i].AIError = describeAIError(err)
				s.deps.Logger.Warn("gemini explanation fallback", "meter_id", w.meter.MeterID, "error", err)
				return nil
			}
			anomalies[i] = mergeExplanation(anomalies[i], exp, s.deps.ConfidenceShift)
			return nil
		})
	}
	_ = g.Wait()

	ok, disabled := 0, 0
	var firstErr error
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrExplainerDisabled):
			disabled++
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	n := len(work)
	switch {
	case ok == n:
		return anomalies, AIStatusOK, fmt.Sprintf("%s explicó %d/%d anomalías con evidencia", s.deps.Explainer.Name(), ok, n)
	case disabled == n:
		return anomalies, AIStatusDisabled, "Explicaciones generadas por el motor de reglas (Gemini no configurado)"
	case ok == 0:
		return anomalies, AIStatusFallback, "Fallback a reglas: " + describeAIError(firstErr)
	}
	return anomalies, AIStatusPartial, fmt.Sprintf("%s explicó %d/%d anomalías · fallback a reglas en %d", s.deps.Explainer.Name(), ok, n, n-ok)
}

func mergeExplanation(a anomaly.Anomaly, exp Explanation, maxShift float64) anomaly.Anomaly {
	if reason := strings.TrimSpace(exp.Reason); reason != "" {
		a.Reason = reason
	}
	if action := strings.TrimSpace(exp.RecommendedAction); action != "" {
		a.RecommendedAction = action
	}
	if exp.Confidence > 0 {
		c := math.Max(a.Confidence-maxShift, math.Min(a.Confidence+maxShift, exp.Confidence))
		a.Confidence = analytics.Round(math.Max(0.05, math.Min(0.99, c)), 2)
	}
	insights := make([]string, 0, len(exp.Insights))
	for _, in := range exp.Insights {
		if trimmed := strings.TrimSpace(in); trimmed != "" {
			insights = append(insights, trimmed)
		}
	}
	a.AIInsights = insights
	a.Source = anomaly.SourceGemini
	a.AIError = ""
	return a
}

func describeAIError(err error) string {
	switch {
	case errors.Is(err, ErrExplainerRateLimited):
		return "Gemini alcanzó el límite de peticiones (429); se usó la explicación del motor de reglas."
	case errors.Is(err, ErrExplainerTimeout):
		return "Gemini excedió el tiempo de respuesta; se usó la explicación del motor de reglas."
	case errors.Is(err, ErrExplainerInvalidResponse):
		return "Gemini devolvió una respuesta inválida; se usó la explicación del motor de reglas."
	case errors.Is(err, ErrExplainerUnavailable):
		return "Gemini no está disponible (credenciales, modelo o servicio); se usó la explicación del motor de reglas."
	case errors.Is(err, ErrExplainerDisabled):
		return "Gemini no está configurado; se usó la explicación del motor de reglas."
	}
	return "Error inesperado al consultar Gemini; se usó la explicación del motor de reglas."
}
