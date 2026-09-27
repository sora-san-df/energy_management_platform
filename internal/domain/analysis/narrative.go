package analysis

import (
	"fmt"
	"strings"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/anomaly"
	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/event"
)

func (d *Detector) narrative(f Finding) (string, string) {
	switch f.Type {
	case anomaly.TypeRealAnomaly:
		return realNarrative(f)
	case anomaly.TypeExplainable:
		return explainableNarrative(f)
	case anomaly.TypeFalsePositive:
		return falsePositiveNarrative(f)
	case anomaly.TypeDataQuality:
		return d.dataQualityNarrative(f)
	}
	return "", ""
}

func describeEvent(e *event.Event) string {
	return fmt.Sprintf("%s del %s: \"%s\"", e.Type, formatTime(e.Timestamp), e.Description)
}

func realNarrative(f Finding) (string, string) {
	s := f.Shift
	direction, verb := "por encima", "Aumento"
	if s.Direction == DirectionDown {
		direction, verb = "por debajo", "Caída"
	}
	var reason strings.Builder
	if s.Persistent {
		fmt.Fprintf(&reason, "Consumo %s %s del baseline desde el %s sin evento operativo que lo explique (%s/día vs baseline de %s/día).",
			formatPct(f.VariationPct), direction, formatTime(s.Start), formatKWh(f.CurrentDailyKWh), formatKWh(f.BaselineDailyKWh))
	} else {
		fmt.Fprintf(&reason, "%s de consumo de %s respecto al perfil esperado durante %d h desde el %s, sin evento operativo que lo explique.",
			verb, formatPct(s.DeviationPct), s.Hours, formatTime(s.Start))
	}
	if c := f.Electrical; c != nil && c.Significant {
		fmt.Fprintf(&reason, " La corriente pasó de %s A a %s A y el factor de potencia de %s a %s.",
			formatNumber(c.Before.CurrentA, 0), formatNumber(c.During.CurrentA, 0),
			formatNumber(c.Before.PowerFactor, 2), formatNumber(c.During.PowerFactor, 2))
	}
	action := "Investigar el cambio con el responsable del área y monitorear la evolución en las próximas 24 horas."
	if f.Severity == anomaly.SeverityHigh {
		action = "Investigar medidor e instalación de inmediato: inspección en sitio, identificar cargas no registradas o fallas de equipos y confirmar con operación."
		if c := f.Electrical; c != nil && c.PowerFactorDelta <= -0.08 {
			action += fmt.Sprintf(" Revisar compensación de reactiva (factor de potencia en %s).", formatNumber(c.During.PowerFactor, 2))
		}
	}
	return reason.String(), action
}

func explainableNarrative(f Finding) (string, string) {
	s := f.Shift
	reason := fmt.Sprintf("Aumento de %s respecto al baseline desde el %s, coincidente con el evento %s. El cambio es consistente con la nueva carga operativa (%s/día vs %s/día).",
		formatPct(f.VariationPct), formatTime(s.Start), describeEvent(f.RelatedEvent), formatKWh(f.CurrentDailyKWh), formatKWh(f.BaselineDailyKWh))
	action := "Validar con operación que el nuevo nivel de consumo corresponde a la línea productiva y actualizar el baseline del medidor."
	return reason, action
}

func falsePositiveNarrative(f Finding) (string, string) {
	s := f.Shift
	reason := fmt.Sprintf("Caída de %s durante %d h desde el %s, explicada por el evento %s. El consumo volvió a su nivel normal al finalizar la parada.",
		formatPct(s.DeviationPct), s.Hours, formatTime(s.Start), describeEvent(f.RelatedEvent))
	action := "No escalar. Registrar como falso positivo y excluir la ventana de la parada del cálculo del baseline."
	return reason, action
}

func (d *Detector) dataQualityNarrative(f Finding) (string, string) {
	q := f.Quality
	var problems []string
	if q.VoltageOutOfRange > 0 || q.VoltageJumps > 0 {
		problems = append(problems, fmt.Sprintf("voltaje entre %s V y %s V (nominal %s V)", formatNumber(q.VoltageMin, 1), formatNumber(q.VoltageMax, 1), formatNumber(d.cfg.NominalVoltage, 0)))
	}
	if q.PowerFactorJumps > 0 {
		problems = append(problems, fmt.Sprintf("saltos bruscos del factor de potencia (%s–%s)", formatNumber(q.PowerFactorMin, 2), formatNumber(q.PowerFactorMax, 2)))
	}
	if q.EnergyMismatches > 0 {
		problems = append(problems, "energía que no cuadra con V×I×FP")
	}
	reason := fmt.Sprintf("Consumo estable (%s vs baseline) pero con %d lecturas eléctricamente inconsistentes desde el %s: %s.",
		formatSignedPct(f.VariationPct), q.AffectedReadings, formatTime(*q.FirstIssueAt), strings.Join(problems, ", "))
	action := "Validar el medidor: revisar calibración, conexiones de TC/TP y comunicaciones. No usar estas lecturas para facturación ni para el baseline hasta corregirlas."
	return reason, action
}

func (d *Detector) buildEvidence(f Finding) []anomaly.Evidence {
	ev := []anomaly.Evidence{
		{Label: "Baseline diario", Value: formatKWh(f.BaselineDailyKWh), Detail: "Mediana del consumo diario del período"},
		{Label: "Consumo actual", Value: formatKWh(f.CurrentDailyKWh), Detail: "Último día completo"},
		{Label: "Variación vs baseline", Value: formatSignedPct(f.VariationPct)},
	}
	if s := f.Shift; s != nil {
		state := "recuperado"
		if s.Persistent {
			state = "persistente"
		}
		ev = append(ev,
			anomaly.Evidence{Label: "Inicio del cambio", Value: formatTime(s.Start)},
			anomaly.Evidence{Label: "Duración", Value: fmt.Sprintf("%d h (%s)", s.Hours, state), Detail: fmt.Sprintf("%d horas fuera del umbral de ±%s", s.FlaggedHours, formatPct(d.cfg.DeviationThreshold*100))},
			anomaly.Evidence{Label: "Desviación horaria media", Value: formatSignedPct(s.DeviationPct), Detail: fmt.Sprintf("Observado %s kWh/h vs esperado %s kWh/h", formatNumber(s.ObservedKWh, 1), formatNumber(s.ExpectedKWh, 1))},
		)
	}
	if c := f.Electrical; c != nil {
		ev = append(ev,
			anomaly.Evidence{Label: "Corriente", Value: fmt.Sprintf("%s A → %s A", formatNumber(c.Before.CurrentA, 0), formatNumber(c.During.CurrentA, 0)), Detail: formatSignedPct(c.CurrentChangePct)},
			anomaly.Evidence{Label: "Factor de potencia", Value: fmt.Sprintf("%s → %s", formatNumber(c.Before.PowerFactor, 2), formatNumber(c.During.PowerFactor, 2)), Detail: fmt.Sprintf("Δ %s", formatNumber(c.PowerFactorDelta, 2))},
			anomaly.Evidence{Label: "Voltaje", Value: fmt.Sprintf("%s V → %s V", formatNumber(c.Before.VoltageV, 1), formatNumber(c.During.VoltageV, 1)), Detail: formatSignedPct(c.VoltageChangePct)},
		)
	}
	if f.Type == anomaly.TypeDataQuality || f.Quality.AffectedReadings >= d.cfg.MinQualityReadings {
		q := f.Quality
		ev = append(ev,
			anomaly.Evidence{Label: "Lecturas inconsistentes", Value: fmt.Sprintf("%d", q.AffectedReadings)},
			anomaly.Evidence{Label: "Voltaje fuera de rango", Value: fmt.Sprintf("%d", q.VoltageOutOfRange), Detail: fmt.Sprintf("Rango observado %s–%s V", formatNumber(q.VoltageMin, 1), formatNumber(q.VoltageMax, 1))},
			anomaly.Evidence{Label: "Saltos de voltaje", Value: fmt.Sprintf("%d", q.VoltageJumps), Detail: fmt.Sprintf("Variaciones > %s V entre lecturas consecutivas", formatNumber(d.cfg.VoltageJumpV, 0))},
			anomaly.Evidence{Label: "Saltos de factor de potencia", Value: fmt.Sprintf("%d", q.PowerFactorJumps), Detail: fmt.Sprintf("Rango observado %s–%s", formatNumber(q.PowerFactorMin, 2), formatNumber(q.PowerFactorMax, 2))},
			anomaly.Evidence{Label: "Energía incoherente", Value: fmt.Sprintf("%d", q.EnergyMismatches), Detail: "kWh fuera de rango respecto a V×I×FP"},
		)
	}
	if e := f.RelatedEvent; e != nil {
		detail := "No explica el cambio"
		if f.Explained {
			detail = "Explica el cambio observado"
		} else if e.Type == event.TypeDataQuality {
			detail = "Corrobora el problema de calidad de datos"
		}
		ev = append(ev, anomaly.Evidence{Label: "Evento relacionado", Value: fmt.Sprintf("%s · %s", e.Type, formatTime(e.Timestamp)), Detail: e.Description + " — " + detail})
	} else {
		ev = append(ev, anomaly.Evidence{Label: "Eventos operativos", Value: "Ninguno", Detail: "No hay eventos registrados cerca del cambio"})
	}
	return ev
}
