import { AlertOctagon, ArrowRight, BrainCircuit, Clock, Gauge, ShieldCheck, Sparkles, Zap } from 'lucide-react'
import { Link } from 'react-router-dom'
import { RunAnalysisButton } from '../components/analysis/RunAnalysisButton'
import { DailyConsumptionChart } from '../components/charts/DailyConsumptionChart'
import { AnomalyTypeBadge, MeterStatusBadge, SeverityBadge, SourceBadge } from '../components/ui/Badges'
import { ConfidenceBar, EmptyState, ErrorState, Skeleton } from '../components/ui/Feedback'
import { GlassCard, Section } from '../components/ui/GlassCard'
import { KpiCard } from '../components/ui/KpiCard'
import { api } from '../lib/api'
import { useAnalysis } from '../lib/analysis'
import { useAuth } from '../lib/auth'
import { formatCompactKWh, formatConfidence, formatDate, formatKWh, formatNumber, formatPct, formatRelative } from '../lib/format'
import { anomalyActionLabel, anomalyTypeLabel, confidenceLevel, providerLabel } from '../lib/labels'
import type { AnomalyType, DashboardSummary, MeterStatus } from '../lib/types'
import { useApi } from '../lib/useApi'

function greeting() {
  const h = new Date().getHours()
  if (h < 12) return 'Buenos días'
  if (h < 19) return 'Buenas tardes'
  return 'Buenas noches'
}

function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-32" />
        ))}
      </div>
      <Skeleton className="h-96" />
    </div>
  )
}

function PriorityQueue({ data }: { data: DashboardSummary }) {
  if (data.top_anomalies.length === 0) {
    return (
      <EmptyState
        icon={<BrainCircuit className="size-6" aria-hidden />}
        title={data.last_analysis ? 'Sin anomalías abiertas' : 'Aún no hay análisis'}
        description={data.last_analysis ? 'El último análisis no encontró desviaciones relevantes.' : 'Ejecuta el análisis IA para detectar, explicar y priorizar anomalías.'}
        action={!data.last_analysis && <RunAnalysisButton size="sm" />}
      />
    )
  }
  return (
    <ol className="space-y-2">
      {data.top_anomalies.map((a) => (
        <li key={a.id}>
          <Link
            to={`/anomalies/${a.id}`}
            className="group flex items-center gap-3 rounded-xl border border-white/10 bg-white/[0.03] p-3 transition hover:border-white/25 hover:bg-white/[0.07] focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]"
          >
            <span className="tabular flex size-8 shrink-0 items-center justify-center rounded-lg bg-white/10 text-sm font-bold">{a.priority}</span>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-semibold">{a.meter_id}</span>
                <SeverityBadge severity={a.severity} />
              </div>
              <p className="mt-0.5 truncate text-xs text-ink-soft/80">
                {anomalyTypeLabel[a.type]} · {anomalyActionLabel[a.type]}
              </p>
            </div>
            <ConfidenceBar value={a.confidence} className="hidden sm:flex xl:hidden 2xl:flex" />
            <ArrowRight className="size-4 text-ink-muted transition group-hover:translate-x-0.5 group-hover:text-white" aria-hidden />
          </Link>
        </li>
      ))}
    </ol>
  )
}

const statusOrder: MeterStatus[] = ['CRITICAL', 'ALERT', 'OK', 'PENDING']
const typeOrder: AnomalyType[] = ['REAL_ANOMALY', 'DATA_QUALITY', 'EXPLAINABLE_ANOMALY', 'FALSE_POSITIVE']

export function DashboardPage() {
  const { user } = useAuth()
  const { completedVersion } = useAnalysis()
  const { data, error, loading, reload } = useApi(() => api.dashboard(), [completedVersion])

  if (loading && !data) return <DashboardSkeleton />
  if (error || !data) return <ErrorState message={error?.message ?? 'No se pudo cargar el dashboard'} onRetry={reload} />

  const last = data.last_analysis
  const topAnomaly = data.top_anomalies[0]

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-sm text-ink-soft">
            {greeting()}, {user?.name?.split(' ')[0]}
          </p>
          <h1 className="mt-1 text-2xl font-bold tracking-tight text-shadow-glass sm:text-3xl">Centro de control energético</h1>
          <p className="mt-1 text-sm text-ink-soft/80">
            Periodo {formatDate(data.period_start)} – {formatDate(data.period_end)} · {formatNumber(data.readings_total)} lecturas horarias
          </p>
        </div>
      </div>

      {topAnomaly && topAnomaly.severity === 'HIGH' && (
        <GlassCard className="flex flex-col gap-4 border-status-critical/40 bg-gradient-to-r from-red-500/15 to-transparent p-5 sm:flex-row sm:items-center">
          <span className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-status-critical/20">
            <AlertOctagon className="size-6 text-status-critical" aria-hidden />
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-red-200">Investigar primero · prioridad 1</p>
            <p className="mt-1 font-semibold text-white">
              {topAnomaly.meter_id} · {anomalyTypeLabel[topAnomaly.type]} · {formatPct(topAnomaly.variation_pct)} vs baseline
            </p>
            <p className="mt-0.5 line-clamp-2 text-sm text-ink-soft">{topAnomaly.reason}</p>
          </div>
          <Link
            to={`/anomalies/${topAnomaly.id}`}
            className="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-xl border border-white/20 bg-white/[0.1] px-4 text-sm font-semibold hover:bg-white/[0.16] focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]"
          >
            Ver investigación <ArrowRight className="size-4" aria-hidden />
          </Link>
        </GlassCard>
      )}

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <KpiCard
          label="Medidores"
          value={data.meters_total}
          icon={<Gauge className="size-5" aria-hidden />}
          hint={`${data.meters_by_status.CRITICAL ?? 0} críticos · ${data.meters_by_status.ALERT ?? 0} en alerta · ${data.meters_by_status.OK ?? 0} normales`}
        />
        <KpiCard
          label="Consumo del período"
          value={formatCompactKWh(data.total_consumption_kwh)}
          icon={<Zap className="size-5" aria-hidden />}
          accent="purple"
          hint={`Último día ${formatKWh(data.current_daily_kwh)} · ${formatPct(data.variation_pct)} vs baseline`}
        />
        <KpiCard
          label="Anomalías IA"
          value={last ? `${data.anomalies_detected} detectadas` : '—'}
          icon={<BrainCircuit className="size-5" aria-hidden />}
          accent="purple"
          hint={last ? typeOrder.filter((t) => data.anomalies_by_type[t]).map((t) => `${data.anomalies_by_type[t]} ${anomalyTypeLabel[t].toLowerCase()}`).join(' · ') : 'Ejecuta el análisis para detectar anomalías'}
        />
        <KpiCard
          label="Alta prioridad"
          value={last ? data.high_priority : '—'}
          icon={<AlertOctagon className="size-5" aria-hidden />}
          accent="critical"
          hint={last ? 'Requieren atención prioritaria' : 'Sin datos todavía'}
        />
        <KpiCard
          label="Confianza IA"
          value={data.anomalies_detected ? formatConfidence(data.avg_confidence) : '—'}
          icon={<ShieldCheck className="size-5" aria-hidden />}
          accent="good"
          hint={data.anomalies_detected ? `Nivel ${confidenceLevel(data.avg_confidence).toLowerCase()} · promedio de ${data.anomalies_detected} hallazgos` : 'Métrica agregada del análisis'}
        />
        <KpiCard
          label="Último análisis"
          value={<span className="text-2xl">{last ? formatRelative(last.finished_at ?? last.started_at) : 'Nunca'}</span>}
          icon={<Clock className="size-5" aria-hidden />}
          hint={last ? <span className="flex flex-wrap items-center gap-2">{last.status === 'COMPLETED' ? 'Completado' : last.status === 'RUNNING' ? 'En ejecución' : 'Fallido'} · {providerLabel(last.ai_provider)}</span> : 'Pulsa Run AI Analysis'}
        />
      </div>

      <div className="grid gap-6 xl:grid-cols-3">
        <Section title="Consumo total vs baseline" description="Suma diaria de los 12 medidores frente a la suma de sus baselines" className="xl:col-span-2" dense>
          <DailyConsumptionChart data={data.daily_consumption} />
        </Section>
        <Section
          title="Cola de prioridad"
          description="Qué investigar primero"
          dense
          action={
            data.top_anomalies.length > 0 && (
              <Link to="/anomalies" className="text-sm font-semibold text-aurora-cyan hover:underline">
                Ver todas
              </Link>
            )
          }
        >
          <PriorityQueue data={data} />
        </Section>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Section title="Salud de medidores" description="Estado según el último análisis IA" dense>
          <ul className="space-y-3">
            {statusOrder.map((s) => {
              const count = data.meters_by_status[s] ?? 0
              if (!count && s === 'PENDING') return null
              return (
                <li key={s} className="flex items-center gap-3">
                  <div className="w-28">
                    <MeterStatusBadge status={s} />
                  </div>
                  <div className="h-2 flex-1 overflow-hidden rounded-full bg-white/[0.06]">
                    <div className="h-full rounded-full bg-[#0891b2]" style={{ width: `${(count / Math.max(1, data.meters_total)) * 100}%` }} />
                  </div>
                  <span className="tabular w-8 text-right text-sm font-semibold">{count}</span>
                </li>
              )
            })}
          </ul>
          <Link to="/meters" className="mt-5 inline-flex items-center gap-1 text-sm font-semibold text-aurora-cyan hover:underline">
            Gestionar medidores <ArrowRight className="size-4" aria-hidden />
          </Link>
        </Section>
        <Section title="Hallazgos por tipo" description="Clasificación de la IA: real, explicable, calidad de datos o falso positivo" dense>
          {data.anomalies_detected === 0 ? (
            <p className="text-sm text-ink-soft">Sin hallazgos todavía.</p>
          ) : (
            <ul className="space-y-3">
              {typeOrder.map((t) => (
                <li key={t} className="flex items-center justify-between gap-3">
                  <AnomalyTypeBadge type={t} />
                  <span className="tabular text-sm font-semibold">{data.anomalies_by_type[t] ?? 0}</span>
                </li>
              ))}
            </ul>
          )}
          {last && (
            <div className="mt-5 flex flex-wrap items-center gap-2 border-t border-white/10 pt-4 text-xs text-ink-soft">
              <Sparkles className="size-3.5 text-violet-300" aria-hidden />
              {last.summary}
              {data.top_anomalies[0] && <SourceBadge source={data.top_anomalies[0].source} />}
            </div>
          )}
        </Section>
      </div>
    </div>
  )
}
