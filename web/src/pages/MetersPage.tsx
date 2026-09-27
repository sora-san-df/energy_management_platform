import clsx from 'clsx'
import { ArrowDownWideNarrow, ArrowUpNarrowWide, ChevronRight, Gauge, Search, TrendingDown, TrendingUp } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { MeterStatusBadge, SeverityBadge } from '../components/ui/Badges'
import { EmptyState, ErrorState, Skeleton } from '../components/ui/Feedback'
import { GlassCard } from '../components/ui/GlassCard'
import { api } from '../lib/api'
import { useAnalysis } from '../lib/analysis'
import { formatKWh, formatPct } from '../lib/format'
import { anomalyTypeLabel } from '../lib/labels'
import { useApi } from '../lib/useApi'

const filters = [
  { value: 'all', label: 'Todos' },
  { value: 'normal', label: 'Normales' },
  { value: 'alert', label: 'Alertas' },
  { value: 'critical', label: 'Críticos' },
]

const sorts = [
  { value: 'meter_id', label: 'Medidor' },
  { value: 'consumption', label: 'Consumo' },
  { value: 'variation', label: 'Variación' },
  { value: 'severity', label: 'Severidad' },
]

function Variation({ value }: { value: number }) {
  const big = Math.abs(value) >= 20
  const Icon = value >= 0 ? TrendingUp : TrendingDown
  return (
    <span className={clsx('tabular inline-flex items-center gap-1 font-semibold', big ? 'text-white' : 'text-ink-soft')}>
      {big && <Icon className={clsx('size-4', value > 0 ? 'text-status-warning' : 'text-status-info')} aria-hidden />}
      {formatPct(value)}
    </span>
  )
}

export function MetersPage() {
  const [params, setParams] = useSearchParams()
  const navigate = useNavigate()
  const { completedVersion } = useAnalysis()
  const status = params.get('status') ?? 'all'
  const sort = params.get('sort') ?? 'meter_id'
  const order = params.get('order') ?? (sort === 'meter_id' ? 'asc' : 'desc')
  const [search, setSearch] = useState(params.get('search') ?? '')
  const [debounced, setDebounced] = useState(search)

  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(search.trim()), 250)
    return () => window.clearTimeout(t)
  }, [search])

  const update = (key: string, value: string | null) => {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  const { data, error, loading, reload } = useApi(() => api.meters({ status, sort, order, search: debounced }), [status, sort, order, debounced, completedVersion])
  const { data: all } = useApi(() => api.meters(), [completedVersion])
  const counts = {
    all: all?.total ?? 0,
    normal: all?.items.filter((m) => m.status === 'OK').length ?? 0,
    alert: all?.items.filter((m) => m.status === 'ALERT').length ?? 0,
    critical: all?.items.filter((m) => m.status === 'CRITICAL').length ?? 0,
  }
  const pending = all?.items.some((m) => m.status === 'PENDING')

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-shadow-glass sm:text-3xl">Gestión de medidores</h1>
        <p className="mt-1 text-sm text-ink-soft/80">Consumo actual (último día completo) frente al baseline histórico de cada medidor.</p>
      </div>

      <GlassCard dense className="p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-wrap gap-2" role="tablist" aria-label="Filtrar por estado">
            {filters.map((f) => (
              <button
                key={f.value}
                role="tab"
                aria-selected={status === f.value}
                onClick={() => update('status', f.value === 'all' ? null : f.value)}
                className={clsx(
                  'inline-flex items-center gap-2 rounded-xl px-3.5 py-2 text-sm font-semibold transition focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]',
                  status === f.value ? 'bg-white/[0.14] text-white shadow-[inset_0_0_0_1px_rgba(255,255,255,0.2)]' : 'text-ink-soft hover:bg-white/[0.06]',
                )}
              >
                {f.label}
                <span className="tabular rounded-md bg-white/10 px-1.5 text-xs">{counts[f.value as keyof typeof counts]}</span>
              </button>
            ))}
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <label className="relative">
              <span className="sr-only">Buscar medidor</span>
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-muted" aria-hidden />
              <input
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value)
                  update('search', e.target.value || null)
                }}
                placeholder="Buscar por meter_id o nombre"
                className="h-10 w-full rounded-xl border border-white/20 bg-white/[0.06] pl-9 pr-3 text-sm text-white placeholder:text-ink-muted focus:outline-none focus:ring-2 focus:ring-[#06B6D4] sm:w-64"
              />
            </label>
            <div className="flex gap-2">
              <label className="flex-1">
                <span className="sr-only">Ordenar por</span>
                <select
                  value={sort}
                  onChange={(e) => update('sort', e.target.value)}
                  className="h-10 w-full rounded-xl border border-white/20 bg-[#0F172A] px-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-[#06B6D4]"
                >
                  {sorts.map((s) => (
                    <option key={s.value} value={s.value}>
                      Ordenar: {s.label}
                    </option>
                  ))}
                </select>
              </label>
              <button
                onClick={() => update('order', order === 'asc' ? 'desc' : 'asc')}
                className="flex size-10 items-center justify-center rounded-xl border border-white/20 bg-white/[0.06] hover:bg-white/[0.12] focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]"
                aria-label={order === 'asc' ? 'Orden ascendente' : 'Orden descendente'}
              >
                {order === 'asc' ? <ArrowUpNarrowWide className="size-4" aria-hidden /> : <ArrowDownWideNarrow className="size-4" aria-hidden />}
              </button>
            </div>
          </div>
        </div>
        {pending && <p className="mt-3 text-xs text-ink-soft">Los medidores aparecen como “Sin analizar” hasta ejecutar el primer Run AI Analysis.</p>}
      </GlassCard>

      {error ? (
        <ErrorState message={error.message} onRetry={reload} />
      ) : (
        <GlassCard dense className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[820px] text-left text-sm">
              <thead className="border-b border-white/10 text-xs uppercase tracking-wide text-ink-muted">
                <tr>
                  <th scope="col" className="px-5 py-3 font-semibold">Medidor</th>
                  <th scope="col" className="px-5 py-3 text-right font-semibold">Consumo actual</th>
                  <th scope="col" className="px-5 py-3 text-right font-semibold">Baseline</th>
                  <th scope="col" className="px-5 py-3 text-right font-semibold">Variación</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Estado</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Anomalía</th>
                  <th scope="col" className="w-10 px-5 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.06]">
                {loading && !data
                  ? Array.from({ length: 6 }).map((_, i) => (
                      <tr key={i}>
                        <td colSpan={7} className="px-5 py-3">
                          <Skeleton className="h-8" />
                        </td>
                      </tr>
                    ))
                  : data?.items.map((m) => (
                      <tr
                        key={m.meter_id}
                        onClick={() => navigate(`/meters/${m.meter_id}`)}
                        className="cursor-pointer transition hover:bg-white/[0.05]"
                      >
                        <td className="px-5 py-3.5">
                          <Link to={`/meters/${m.meter_id}`} className="font-semibold text-white hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]" onClick={(e) => e.stopPropagation()}>
                            {m.meter_id}
                          </Link>
                          <p className="text-xs text-ink-soft/80">
                            {m.name} · {m.location}
                          </p>
                        </td>
                        <td className="tabular px-5 py-3.5 text-right font-semibold">{formatKWh(m.current_daily_kwh)}</td>
                        <td className="tabular px-5 py-3.5 text-right text-ink-soft">{formatKWh(m.baseline_daily_kwh)}</td>
                        <td className="px-5 py-3.5 text-right">
                          <Variation value={m.variation_pct} />
                        </td>
                        <td className="px-5 py-3.5">
                          <MeterStatusBadge status={m.status} />
                        </td>
                        <td className="px-5 py-3.5">
                          {m.anomaly && m.anomaly.type !== 'FALSE_POSITIVE' ? (
                            <div className="flex items-center gap-2">
                              <SeverityBadge severity={m.anomaly.severity} />
                              <span className="hidden text-xs text-ink-soft xl:inline">{anomalyTypeLabel[m.anomaly.type]}</span>
                            </div>
                          ) : m.anomaly ? (
                            <span className="text-xs text-ink-soft">Falso positivo · no escalar</span>
                          ) : (
                            <span className="text-ink-muted">—</span>
                          )}
                        </td>
                        <td className="px-5 py-3.5 text-ink-muted">
                          <ChevronRight className="size-4" aria-hidden />
                        </td>
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>
          {data && data.items.length === 0 && (
            <EmptyState icon={<Gauge className="size-6" aria-hidden />} title="Sin resultados" description="Ningún medidor coincide con los filtros seleccionados." />
          )}
        </GlassCard>
      )}
    </div>
  )
}
