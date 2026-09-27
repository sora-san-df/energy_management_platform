import clsx from 'clsx'
import { ArrowLeft, ArrowRight, CalendarClock, Gauge, MapPin, Sigma, TrendingUp, Zap } from 'lucide-react'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { RunAnalysisButton } from '../components/analysis/RunAnalysisButton'
import { DailyConsumptionChart } from '../components/charts/DailyConsumptionChart'
import { ElectricalCharts } from '../components/charts/ElectricalCharts'
import { HourlyConsumptionChart } from '../components/charts/HourlyConsumptionChart'
import { AnomalyTypeBadge, MeterStatusBadge, SeverityBadge, SourceBadge } from '../components/ui/Badges'
import { ConfidenceBar, ErrorState, Skeleton } from '../components/ui/Feedback'
import { GlassCard, Section } from '../components/ui/GlassCard'
import { KpiCard } from '../components/ui/KpiCard'
import { api } from '../lib/api'
import { useAnalysis } from '../lib/analysis'
import { formatDateTime, formatKWh, formatNumber, formatPct } from '../lib/format'
import { eventTypeLabel } from '../lib/labels'
import { useApi } from '../lib/useApi'

export function MeterDetailPage() {
  const { meterId = '' } = useParams()
  const { completedVersion } = useAnalysis()
  const [granularity, setGranularity] = useState<'hour' | 'day'>('hour')
  const detail = useApi(() => api.meter(meterId), [meterId, completedVersion])
  const hourly = useApi(() => api.readings(meterId, 'hour'), [meterId])

  if (detail.error) {
    return <ErrorState message={detail.error.status === 404 ? `El medidor ${meterId} no existe` : detail.error.message} onRetry={detail.reload} />
  }
  const m = detail.data
  if (!m) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-20" />
        <div className="grid gap-4 sm:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-28" />
          ))}
        </div>
        <Skeleton className="h-96" />
      </div>
    )
  }

  const anomaly = m.anomalies[0]
  const points = hourly.data?.points ?? []

  return (
    <div className="space-y-6">
      <Link to="/meters" className="inline-flex items-center gap-1.5 text-sm text-ink-soft hover:text-white">
        <ArrowLeft className="size-4" aria-hidden /> Medidores
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-bold tracking-tight text-shadow-glass sm:text-3xl">{m.meter_id}</h1>
            <MeterStatusBadge status={m.status} />
          </div>
          <p className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-ink-soft">
            <span className="font-semibold text-white">{m.name}</span>
            <span className="inline-flex items-center gap-1">
              <MapPin className="size-3.5" aria-hidden /> {m.location}
            </span>
            <span className="inline-flex items-center gap-1">
              <CalendarClock className="size-3.5" aria-hidden /> Última lectura {formatDateTime(m.last_reading_at)}
            </span>
          </p>
        </div>
        <RunAnalysisButton meterIds={[m.meter_id]} label={`Analizar ${m.meter_id} con IA`} />
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard label="Consumo actual" value={formatKWh(m.current_daily_kwh)} icon={<Zap className="size-5" aria-hidden />} hint="Último día completo (24 h)" />
        <KpiCard label="Baseline histórico" value={formatKWh(m.baseline_daily_kwh)} icon={<Sigma className="size-5" aria-hidden />} accent="purple" hint="Mediana del consumo diario" />
        <KpiCard
          label="Variación"
          value={formatPct(m.variation_pct)}
          icon={<TrendingUp className="size-5" aria-hidden />}
          accent={Math.abs(m.variation_pct) >= 50 ? 'critical' : Math.abs(m.variation_pct) >= 20 ? 'warning' : 'good'}
          hint="Consumo actual vs baseline"
        />
        <KpiCard label="Total del período" value={formatKWh(m.total_kwh)} icon={<Gauge className="size-5" aria-hidden />} hint={`${formatNumber(m.readings_count)} lecturas`} />
      </div>

      {anomaly && (
        <GlassCard
          className={clsx(
            'flex flex-col gap-4 p-5 lg:flex-row lg:items-center',
            anomaly.severity === 'HIGH' ? 'border-status-critical/40' : anomaly.type === 'FALSE_POSITIVE' ? '' : 'border-status-warning/40',
          )}
        >
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <AnomalyTypeBadge type={anomaly.type} />
              <SeverityBadge severity={anomaly.severity} />
              <SourceBadge source={anomaly.source} />
              <span className="text-xs text-ink-soft">Prioridad #{anomaly.priority}</span>
            </div>
            <p className="mt-3 text-sm text-white">{anomaly.reason}</p>
            <p className="mt-2 text-sm text-ink-soft">
              <span className="font-semibold text-aurora-cyan">Acción: </span>
              {anomaly.recommended_action}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-4">
            <ConfidenceBar value={anomaly.confidence} />
            <Link
              to={`/anomalies/${anomaly.id}`}
              className="inline-flex h-10 items-center gap-2 rounded-xl border border-white/20 bg-white/[0.1] px-4 text-sm font-semibold hover:bg-white/[0.16] focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]"
            >
              Investigar <ArrowRight className="size-4" aria-hidden />
            </Link>
          </div>
        </GlassCard>
      )}

      <Section
        title="Consumo actual vs baseline"
        description={granularity === 'hour' ? 'Consumo horario frente al perfil esperado por hora del día' : 'Consumo diario frente al baseline histórico'}
        dense
        action={
          <div className="flex rounded-xl border border-white/15 bg-white/[0.04] p-1" role="group" aria-label="Granularidad">
            {(['hour', 'day'] as const).map((g) => (
              <button
                key={g}
                onClick={() => setGranularity(g)}
                aria-pressed={granularity === g}
                className={clsx('rounded-lg px-3 py-1 text-xs font-semibold transition', granularity === g ? 'bg-white/[0.14] text-white' : 'text-ink-soft hover:text-white')}
              >
                {g === 'hour' ? 'Por hora' : 'Por día'}
              </button>
            ))}
          </div>
        }
      >
        {granularity === 'hour' ? (
          hourly.loading && !hourly.data ? (
            <Skeleton className="h-80" />
          ) : (
            <HourlyConsumptionChart
              points={points}
              events={m.events}
              window={anomaly && anomaly.type !== 'DATA_QUALITY' ? { start: anomaly.window_start, end: anomaly.window_end, label: 'Ventana de la anomalía' } : undefined}
            />
          )
        ) : (
          <DailyConsumptionChart data={m.daily} />
        )}
      </Section>

      <Section title="Variables eléctricas" description="Voltaje (banda nominal 220 V ±5%), corriente y factor de potencia por hora" dense>
        {hourly.data ? <ElectricalCharts points={points} highlight={anomaly?.type !== 'FALSE_POSITIVE' ? anomaly?.affected_variables : undefined} /> : <Skeleton className="h-44" />}
      </Section>

      <div className="grid gap-6 lg:grid-cols-2">
        <Section title="Promedios del período" dense>
          <dl className="grid grid-cols-2 gap-4">
            {[
              ['Consumo por hora', `${formatNumber(m.electrical.consumption_kwh, 1)} kWh`],
              ['Voltaje', `${formatNumber(m.electrical.voltage_v, 1)} V`],
              ['Corriente', `${formatNumber(m.electrical.current_a, 1)} A`],
              ['Factor de potencia', formatNumber(m.electrical.power_factor, 3)],
            ].map(([label, value]) => (
              <div key={label} className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
                <dt className="text-xs text-ink-muted">{label}</dt>
                <dd className="tabular mt-1 text-lg font-semibold text-white">{value}</dd>
              </div>
            ))}
          </dl>
        </Section>
        <Section title="Eventos operativos" description="Registrados para este medidor" dense>
          {m.events.length === 0 ? (
            <p className="text-sm text-ink-soft">Sin eventos registrados en el período.</p>
          ) : (
            <ul className="space-y-3">
              {m.events.map((e) => (
                <li key={e.id} className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="text-sm font-semibold text-white">{eventTypeLabel[e.type] ?? e.type}</span>
                    <span className="text-xs text-ink-muted">{formatDateTime(e.timestamp)}</span>
                  </div>
                  <p className="mt-1 text-sm text-ink-soft">{e.description}</p>
                </li>
              ))}
            </ul>
          )}
        </Section>
      </div>
    </div>
  )
}
