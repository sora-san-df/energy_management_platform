export type MeterStatus = 'PENDING' | 'OK' | 'ALERT' | 'CRITICAL'
export type AnomalyType = 'REAL_ANOMALY' | 'EXPLAINABLE_ANOMALY' | 'FALSE_POSITIVE' | 'DATA_QUALITY'
export type Severity = 'HIGH' | 'MEDIUM' | 'LOW'
export type AnomalyStatus = 'OPEN' | 'INVESTIGATING' | 'RESOLVED' | 'DISMISSED'
export type RunStatus = 'RUNNING' | 'COMPLETED' | 'FAILED'
export type StepStatus = 'PENDING' | 'RUNNING' | 'DONE' | 'FAILED'
export type AIStatus = 'ok' | 'partial' | 'fallback' | 'disabled' | ''

export interface User {
  email: string
  name: string
  role: string
}

export interface Session {
  access_token: string
  token_type: string
  expires_at: string
  user: User
}

export interface ListResponse<T> {
  items: T[]
  total: number
}

export interface EnergyEvent {
  id: number
  meter_id: string
  timestamp: string
  type: string
  description: string
}

export interface Evidence {
  label: string
  value: string
  detail?: string
}

export interface Anomaly {
  id: number
  analysis_id: number
  meter_id: string
  detected_at: string
  type: AnomalyType
  severity: Severity
  confidence: number
  reason: string
  recommended_action: string
  status: AnomalyStatus
  priority: number
  source: 'gemini' | 'rules'
  ai_error?: string
  ai_insights: string[]
  baseline_daily_kwh: number
  current_daily_kwh: number
  variation_pct: number
  window_start?: string
  window_end?: string
  affected_variables: string[]
  evidence: Evidence[]
  related_event?: EnergyEvent
  created_at: string
}

export interface AnomalyRef {
  id: number
  type: AnomalyType
  severity: Severity
  confidence: number
  status: AnomalyStatus
}

export interface MeterOverview {
  id: number
  meter_id: string
  name: string
  location: string
  status: MeterStatus
  created_at: string
  total_kwh: number
  current_daily_kwh: number
  baseline_daily_kwh: number
  variation_pct: number
  readings_count: number
  last_reading_at?: string
  anomaly?: AnomalyRef
}

export interface ElectricalAverages {
  consumption_kwh: number
  voltage_v: number
  current_a: number
  power_factor: number
}

export interface Reading {
  id: number
  meter_id: string
  timestamp: string
  consumption_kwh: number
  voltage_v: number
  current_a: number
  power_factor: number
  status: string
}

export interface DailyPoint {
  date: string
  kwh: number
  baseline_kwh: number
}

export interface MeterDetail extends MeterOverview {
  electrical: ElectricalAverages
  latest_reading?: Reading
  daily: DailyPoint[]
  events: EnergyEvent[]
  anomalies: Anomaly[]
}

export interface SeriesPoint {
  timestamp: string
  consumption_kwh: number
  expected_kwh: number
  voltage_v: number
  current_a: number
  power_factor: number
  status?: string
}

export interface ReadingsResult {
  meter_id: string
  granularity: 'hour' | 'day'
  points: SeriesPoint[]
}

export interface Step {
  key: string
  label: string
  status: StepStatus
  detail?: string
  started_at?: string
  finished_at?: string
}

export interface AnalysisRun {
  id: number
  status: RunStatus
  scope: string[]
  steps: Step[]
  current_step: string
  meters_analyzed: number
  anomalies_detected: number
  high_priority: number
  avg_confidence: number
  ai_provider: string
  ai_status: AIStatus
  summary: string
  error?: string
  duration_ms: number
  started_at: string
  finished_at?: string
}

export interface DashboardSummary {
  meters_total: number
  meters_by_status: Partial<Record<MeterStatus, number>>
  readings_total: number
  total_consumption_kwh: number
  baseline_daily_kwh: number
  current_daily_kwh: number
  variation_pct: number
  period_start?: string
  period_end?: string
  anomalies_detected: number
  high_priority: number
  avg_confidence: number
  confidence_level: string
  anomalies_by_type: Partial<Record<AnomalyType, number>>
  top_anomalies: Anomaly[]
  daily_consumption: DailyPoint[]
  last_analysis: AnalysisRun | null
}
