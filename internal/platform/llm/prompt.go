package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/analysis"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
)

const systemPrompt = `Eres un analista senior de gestión energética industrial. Recibes un hallazgo producido por un motor determinista de detección de anomalías sobre medidores eléctricos (consumo kWh, voltaje V, corriente A y factor de potencia).

Tu tarea es redactar, en español neutro y profesional, una explicación y una recomendación operativa sustentadas únicamente en la evidencia entregada.

Reglas:
- No cambies el tipo ni la severidad del hallazgo: ya fueron decididos por el motor determinista.
- Usa solo los datos del hallazgo; no inventes cifras, eventos ni equipos.
- La explicación ("reason") debe tener 1 a 3 frases, citar las cifras clave (variación vs baseline, fechas, variables eléctricas) y justificar por qué el hallazgo es del tipo indicado.
- La recomendación ("recommended_action") debe ser una acción concreta, priorizada y coherente con el tipo: REAL_ANOMALY → investigar en sitio; EXPLAINABLE_ANOMALY → validar con operación y ajustar baseline; FALSE_POSITIVE → no escalar; DATA_QUALITY → validar el medidor antes de usar los datos.
- "confidence" es tu confianza (0 a 1) en que la clasificación es correcta según la evidencia.
- "insights" son 2 a 4 observaciones breves que respalden la conclusión.
- Responde exclusivamente con un objeto JSON válido con las claves: reason, recommended_action, confidence, insights.`

var typeDescriptions = map[anomaly.Type]string{
	anomaly.TypeRealAnomaly:   "Anomalía real: desviación significativa sin evento operativo que la explique.",
	anomaly.TypeExplainable:   "Anomalía explicable: el cambio coincide con un evento operativo conocido.",
	anomaly.TypeFalsePositive: "Falso positivo: el cambio está explicado por un evento programado y no debe escalarse.",
	anomaly.TypeDataQuality:   "Problema de calidad de datos: lecturas eléctricas inconsistentes con el consumo registrado.",
}

type promptEvent struct {
	Type        event.Type `json:"type"`
	Timestamp   time.Time  `json:"timestamp"`
	Description string     `json:"description"`
	Explains    bool       `json:"explains_change"`
}

type promptPayload struct {
	Meter struct {
		ID       string `json:"meter_id"`
		Name     string `json:"name"`
		Location string `json:"location"`
	} `json:"meter"`
	Classification struct {
		Type        anomaly.Type     `json:"type"`
		Description string           `json:"description"`
		Severity    anomaly.Severity `json:"severity"`
		Confidence  float64          `json:"engine_confidence"`
	} `json:"classification"`
	Consumption struct {
		BaselineDailyKWh float64 `json:"baseline_daily_kwh"`
		CurrentDailyKWh  float64 `json:"current_daily_kwh"`
		VariationPct     float64 `json:"variation_pct"`
	} `json:"consumption"`
	Shift             *analysis.Shift            `json:"consumption_shift,omitempty"`
	Electrical        *analysis.ElectricalChange `json:"electrical_change,omitempty"`
	Quality           *analysis.QualityReport    `json:"data_quality,omitempty"`
	RelatedEvent      *promptEvent               `json:"related_event,omitempty"`
	AffectedVariables []string                   `json:"affected_variables"`
	Evidence          []anomaly.Evidence         `json:"evidence"`
	EngineReason      string                     `json:"engine_reason"`
	EngineAction      string                     `json:"engine_recommended_action"`
}

func BuildPrompt(req analysis.ExplainRequest) (string, error) {
	f := req.Finding
	var p promptPayload
	p.Meter.ID, p.Meter.Name, p.Meter.Location = f.MeterID, req.MeterName, req.Location
	p.Classification.Type = f.Type
	p.Classification.Description = typeDescriptions[f.Type]
	p.Classification.Severity = f.Severity
	p.Classification.Confidence = f.Confidence
	p.Consumption.BaselineDailyKWh = f.BaselineDailyKWh
	p.Consumption.CurrentDailyKWh = f.CurrentDailyKWh
	p.Consumption.VariationPct = f.VariationPct
	p.Shift = f.Shift
	p.Electrical = f.Electrical
	if f.Quality.AffectedReadings > 0 {
		q := f.Quality
		p.Quality = &q
	}
	if e := f.RelatedEvent; e != nil {
		p.RelatedEvent = &promptEvent{Type: e.Type, Timestamp: e.Timestamp, Description: e.Description, Explains: f.Explained}
	}
	p.AffectedVariables = f.AffectedVariables
	p.Evidence = f.Evidence
	p.EngineReason = f.Reason
	p.EngineAction = f.RecommendedAction

	body, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode prompt: %w", err)
	}
	return "Hallazgo del motor de anomalías:\n" + string(body) + "\n\nGenera la explicación y la recomendación en el formato JSON indicado.", nil
}

type rawExplanation struct {
	Reason            string   `json:"reason"`
	RecommendedAction string   `json:"recommended_action"`
	Confidence        *float64 `json:"confidence"`
	Insights          []string `json:"insights"`
}

func stripFences(text string) string {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, "```") {
		t = strings.TrimPrefix(t, "```json")
		t = strings.TrimPrefix(t, "```")
		t = strings.TrimSuffix(strings.TrimSpace(t), "```")
	}
	return strings.TrimSpace(t)
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{analysis.ErrExplainerInvalidResponse}, args...)...)
}

func ParseExplanation(text string) (analysis.Explanation, error) {
	clean := stripFences(text)
	if clean == "" {
		return analysis.Explanation{}, invalid("empty response")
	}
	var raw rawExplanation
	if err := json.Unmarshal([]byte(clean), &raw); err != nil {
		return analysis.Explanation{}, invalid("decode json: %v", err)
	}
	exp := analysis.Explanation{
		Reason:            strings.TrimSpace(raw.Reason),
		RecommendedAction: strings.TrimSpace(raw.RecommendedAction),
		Insights:          []string{},
	}
	if exp.Reason == "" || exp.RecommendedAction == "" {
		return analysis.Explanation{}, invalid("missing reason or recommended_action")
	}
	if raw.Confidence != nil {
		c := *raw.Confidence
		if c > 1 && c <= 100 {
			c /= 100
		}
		if c < 0 || c > 1 {
			return analysis.Explanation{}, invalid("confidence out of range: %v", *raw.Confidence)
		}
		exp.Confidence = c
	}
	for _, in := range raw.Insights {
		if trimmed := strings.TrimSpace(in); trimmed != "" {
			exp.Insights = append(exp.Insights, trimmed)
		}
	}
	return exp, nil
}
