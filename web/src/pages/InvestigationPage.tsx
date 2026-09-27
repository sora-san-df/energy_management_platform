import clsx from 'clsx'
import { AlertTriangle, ArrowLeft, CalendarClock, CheckCircle2, ClipboardCheck, Lightbulb, Search, ShieldOff, Sparkles, Target, Workflow } from 'lucide-react'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ElectricalCharts } from '../components/charts/ElectricalCharts'
import { HourlyConsumptionChart } from '../components/charts/HourlyConsumptionChart'
import { AnomalyStatusBadge, AnomalyTypeBadge, Pill, SeverityBadge, SourceBadge } from '../components/ui/Badges'
import { Button } from '../components/ui/Button'
import { ErrorState, Skeleton } from '../components/ui/Feedback'
import { GlassCard, Section } from '../components/ui/GlassCard'
import { ApiError, api } from '../lib/api'
import { useAnalysis } from '../lib/analysis'
import { formatConfidence, formatDateTime, formatKWh, formatPct } from '../lib/format'
import { anomalyTypeHint, confidenceLevel, eventTypeLabel, severityLabel, variableLabel } from '../lib/labels'
import type { Anomaly, AnomalyStatus } from '../lib/types'
import { useApi } from '../lib/useApi'

const statusActions: { status: AnomalyStatus; label: string; icon: typeof Search }[] = [
  { status: 'INVESTIGATING', label: 'Marcar en investigación', icon: Search },
  { status: 'RESOLVED', label: 'Marcar resuelta', icon: CheckCircle2 },
  { status: 'DISMISSED', label: 'Descartar', icon: ShieldOff },
]

function ConfidenceGauge({ value }: { value: number }) {
  const pct = Math.round(value * 100)
  const r = 42
  const c = 2 * Math.PI * r
  return (
    <div className="relative size-32">
      <svg viewBox="0 0 100 100" className="size-full -rotate-90" role="img" aria-label={`Confianza ${pct}%`}>
        <defs>
          <linearGradient id="gauge" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stopColor="#06B6D4" />
            <stop offset="1" stopColor="#8B5CF6" />
          </linearGradient>
        </defs>
        <circle cx="50" cy="50" r={r} fill="none" stroke="rgba(255,255,255,0.08)" strokeWidth="8" />
        <circle cx="50" cy="50" r={r} fill="none" stroke="url(#gauge)" strokeWidth="8" strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - value)} />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="tabular text-2xl font-bold text-white">{pct}%</span>
        <span className="text-xs text-ink-soft">{confidenceLevel(value)}</span>
      </div>
    </div>
  )
}

function Header({ a, onStatus, updating }: { a: Anomaly; onStatus: (s: AnomalyStatus) => void; updating: AnomalyStatus | null }) {
  return (
    <div className="space-y-4">
      <Link to="/anomalies" className="inline-flex items-center gap-1.5 text-sm text-ink-soft hover:text-white">
        <ArrowLeft className="size-4" aria-hidden /> Anomalías IA
      </Link>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.14em] text-aurora-cyan">Investigación · prioridad #{a.priority}</p>
          <h1 className="mt-1 text-2xl font-bold tracking-tight text-shadow-glass sm:text-3xl">
            <Link to={`/meters/${a.meter_id}`} className="hover:underline">
              {a.meter_id}
            </Link>
          </h1>
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <AnomalyTypeBadge type={a.type} />
            <SeverityBadge severity={a.severity} />
            <AnomalyStatusBadge status={a.status} />
            <SourceBadge source={a.source} />
            <span className="inline-flex items-center gap-1 text-xs text-ink-soft">
              <CalendarClock className="size-3.5" aria-hidden /> Detectada {formatDateTime(a.detected_at)}
            </span>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {statusActions
            .filter((s) => s.status !== a.status)
            .map(({ status, label, icon: Icon }) => (
              <Button key={status} size="sm" loading={updating === status} icon={<Icon className="size-3.5" aria-hidden />} onClick={() => onStatus(status)}>
                {label}
              </Button>
            ))}
        </div>
      </div>
    </div>
  )
}

export function InvestigationPage() {
  const { id = '' } = useParams()
  const { completedVersion } = useAnalysis()
  const anomaly = useApi(() => api.anomaly(id), [id, completedVersion])
  const meterId = anomaly.data?.meter_id
  const readings = useApi(() => (meterId ? api.readings(meterId, 'hour') : Promise.resolve(undefined)), [meterId])
  const [override, setOverride] = useState<Anomaly | null>(null)
  const [updating, setUpdating] = useState<AnomalyStatus | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  if (anomaly.error) {
    return <ErrorState message={anomaly.error.status === 404 ? 'Esta anomalía ya no existe. Puede haber sido reemplazada por un análisis más reciente.' : anomaly.error.message} onRetry={anomaly.reload} />
  }
  const a = override && override.id === anomaly.data?.id ? override : anomaly.data
  if (!a) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-24" />
        <div className="grid gap-6 lg:grid-cols-3">
          <Skeleton className="h-64 lg:col-span-2" />
          <Skeleton className="h-64" />
        </div>
      </div>
    )
  }

  async function changeStatus(status: AnomalyStatus) {
    if (!a) return
    setUpdating(status)
    setActionError(null)
    try {
      setOverride(await api.updateAnomalyStatus(a.id, status))
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'No se pudo actualizar el estado')
    } finally {
      setUpdating(null)
    }
  }

  const points = readings.data?.points ?? []
  const related = a.related_event
  const explains = related && (a.type === 'EXPLAINABLE_ANOMALY' || a.type === 'FALSE_POSITIVE')

  return (
    <div className="space-y-6">
      <Header a={a} onStatus={changeStatus} updating={updating} />
      {actionError && (
        <p className="rounded-xl border border-status-critical/40 bg-status-critical/10 px-3 py-2 text-sm text-red-100" role="alert">
          {actionError}
        </p>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <GlassCard dense className="p-6 lg:col-span-2">
          <div className="flex items-center gap-2 text-violet-200">
            <Sparkles className="size-4" aria-hidden />
            <h2 className="text-sm font-semibold uppercase tracking-[0.14em]">Qué encontró la IA</h2>
          </div>
          <p className="mt-3 text-lg leading-relaxed text-white">{a.reason}</p>
          <p className="mt-2 text-sm text-ink-soft/80">{anomalyTypeHint[a.type]}.</p>
          {a.ai_insights.length > 0 && (
            <ul className="mt-5 space-y-2">
              {a.ai_insights.map((insight) => (
                <li key={insight} className="flex gap-2 text-sm text-ink-soft">
                  <Lightbulb className="mt-0.5 size-4 shrink-0 text-aurora-cyan" aria-hidden />
                  {insight}
                </li>
              ))}
            </ul>
          )}
          {a.ai_error && (
            <div className="mt-5 flex gap-2 rounded-xl border border-status-warning/40 bg-status-warning/10 p-3 text-xs text-amber-100">
              <AlertTriangle className="size-4 shrink-0 text-status-warning" aria-hidden />
              {a.ai_error}
            </div>
          )}
          <div className="mt-6 rounded-2xl border border-aurora-cyan/30 bg-gradient-to-r from-[#06B6D4]/15 to-[#8B5CF6]/10 p-4">
            <div className="flex items-center gap-2 text-cyan-100">
              <Target className="size-4" aria-hidden />
              <h3 className="text-sm font-semibold uppercase tracking-[0.14em]">Acción recomendada</h3>
            </div>
            <p className="mt-2 font-semibold text-white">{a.recommended_action}</p>
          </div>
        </GlassCard>

        <GlassCard dense className="flex flex-col items-center gap-4 p-6 text-center">
          <h2 className="self-start text-sm font-semibold uppercase tracking-[0.14em] text-ink-soft">Severidad y confianza</h2>
          <ConfidenceGauge value={a.confidence} />
          <div className="grid w-full grid-cols-2 gap-3 text-left">
            <div className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
              <p className="text-xs text-ink-muted">Severidad</p>
              <p className="mt-1 font-semibold text-white">{severityLabel[a.severity]}</p>
            </div>
            <div className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
              <p className="text-xs text-ink-muted">Prioridad</p>
              <p className="mt-1 font-semibold text-white">#{a.priority}</p>
            </div>
          </div>
          <p className="text-xs text-ink-soft/80">
            Confianza {formatConfidence(a.confidence)}: el motor determinista fija tipo y severidad; {a.source === 'gemini' ? 'Gemini ajusta la confianza (±10 pts) y redacta la explicación.' : 'la explicación proviene del motor de reglas.'}
          </p>
        </GlassCard>
      </div>

      <Section title="Comparación contra baseline" description="Consumo horario real frente al perfil esperado; la franja resalta la ventana del hallazgo" dense>
        <dl className="mb-5 grid gap-3 sm:grid-cols-3">
          {[
            ['Baseline diario', formatKWh(a.baseline_daily_kwh)],
            ['Consumo actual', formatKWh(a.current_daily_kwh)],
            ['Variación', formatPct(a.variation_pct)],
          ].map(([label, value]) => (
            <div key={label} className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
              <dt className="text-xs text-ink-muted">{label}</dt>
              <dd className="tabular mt-1 text-xl font-bold text-white">{value}</dd>
            </div>
          ))}
        </dl>
        {readings.data ? (
          <HourlyConsumptionChart points={points} events={related ? [related] : []} window={{ start: a.window_start, end: a.window_end, label: a.type === 'DATA_QUALITY' ? 'Periodo con lecturas inconsistentes' : 'Ventana de la anomalía' }} />
        ) : (
          <Skeleton className="h-72" />
        )}
      </Section>

      <Section
        title="Variables que cambiaron"
        description="Variables eléctricas afectadas según la correlación del motor"
        dense
        action={
          <div className="flex flex-wrap gap-2">
            {a.affected_variables.map((v) => (
              <Pill key={v} tone="critical">
                {variableLabel[v] ?? v}
              </Pill>
            ))}
          </div>
        }
      >
        {readings.data ? <ElectricalCharts points={points} highlight={a.affected_variables} /> : <Skeleton className="h-44" />}
      </Section>

      <div className="grid gap-6 lg:grid-cols-2">
        <Section title="Eventos relacionados" description="Eventos operativos correlacionados con el hallazgo" dense>
          {related ? (
            <div className={clsx('rounded-xl border p-4', explains ? 'border-aurora-purple/40 bg-aurora-purple/10' : 'border-white/15 bg-white/[0.03]')}>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="inline-flex items-center gap-2 font-semibold text-white">
                  <Workflow className="size-4 text-violet-300" aria-hidden />
                  {eventTypeLabel[related.type] ?? related.type}
                </span>
                <span className="text-xs text-ink-muted">{formatDateTime(related.timestamp)}</span>
              </div>
              <p className="mt-2 text-sm text-ink-soft">{related.description}</p>
              <p className="mt-3 text-xs font-semibold text-white">
                {explains ? '✓ Explica el cambio observado' : a.type === 'DATA_QUALITY' ? 'Corrobora el problema de calidad de datos' : '✕ No explica el cambio: no hay causa operativa conocida'}
              </p>
            </div>
          ) : (
            <p className="text-sm text-ink-soft">No hay eventos operativos registrados cerca del cambio: no existe una causa conocida.</p>
          )}
        </Section>
        <Section title="Evidencia" description="Datos que sustentan la explicación" dense>
          <dl className="divide-y divide-white/[0.06]">
            {a.evidence.map((e) => (
              <div key={e.label} className="flex items-start justify-between gap-4 py-2.5">
                <dt className="text-sm text-ink-soft">
                  {e.label}
                  {e.detail && <span className="block text-xs text-ink-muted">{e.detail}</span>}
                </dt>
                <dd className="tabular shrink-0 text-right text-sm font-semibold text-white">{e.value}</dd>
              </div>
            ))}
          </dl>
          <p className="mt-4 flex items-center gap-2 text-xs text-ink-muted">
            <ClipboardCheck className="size-3.5" aria-hidden /> Análisis #{a.analysis_id}
          </p>
        </Section>
      </div>
    </div>
  )
}
