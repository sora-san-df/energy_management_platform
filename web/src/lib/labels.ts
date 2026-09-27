import type { AnomalyStatus, AnomalyType, MeterStatus, Severity } from './types'

export const anomalyTypeLabel: Record<AnomalyType, string> = {
  REAL_ANOMALY: 'Anomalía real',
  EXPLAINABLE_ANOMALY: 'Anomalía explicable',
  FALSE_POSITIVE: 'Falso positivo',
  DATA_QUALITY: 'Calidad de datos',
}

export const anomalyTypeHint: Record<AnomalyType, string> = {
  REAL_ANOMALY: 'Desviación significativa sin evento operativo que la explique',
  EXPLAINABLE_ANOMALY: 'Cambio que coincide con un evento operativo conocido',
  FALSE_POSITIVE: 'Cambio explicado por un evento programado; no se escala',
  DATA_QUALITY: 'Lecturas eléctricas inconsistentes con el consumo registrado',
}

export const anomalyActionLabel: Record<AnomalyType, string> = {
  REAL_ANOMALY: 'Investigar',
  DATA_QUALITY: 'Validar medidor',
  EXPLAINABLE_ANOMALY: 'Validar operación',
  FALSE_POSITIVE: 'No escalar',
}

export const severityLabel: Record<Severity, string> = {
  HIGH: 'Alta',
  MEDIUM: 'Media',
  LOW: 'Baja',
}

export const meterStatusLabel: Record<MeterStatus, string> = {
  OK: 'Normal',
  ALERT: 'Alerta',
  CRITICAL: 'Crítico',
  PENDING: 'Sin analizar',
}

export const anomalyStatusLabel: Record<AnomalyStatus, string> = {
  OPEN: 'Abierta',
  INVESTIGATING: 'En investigación',
  RESOLVED: 'Resuelta',
  DISMISSED: 'Descartada',
}

export const variableLabel: Record<string, string> = {
  consumption_kwh: 'Consumo',
  voltage_v: 'Voltaje',
  current_a: 'Corriente',
  power_factor: 'Factor de potencia',
}

export const eventTypeLabel: Record<string, string> = {
  OPERATIONAL_CHANGE: 'Cambio operativo',
  SCHEDULED_OUTAGE: 'Parada programada',
  DATA_QUALITY: 'Calidad de datos',
  UNKNOWN: 'Sin evento reportado',
}

export function confidenceLevel(value: number) {
  if (value >= 0.85) return 'Alta'
  if (value >= 0.65) return 'Media'
  return 'Baja'
}

export function providerLabel(provider: string) {
  return provider === 'rules' || !provider ? 'Motor de reglas' : `Gemini (${provider})`
}
