import clsx from 'clsx'
import { BrainCircuit, ChevronRight, Sparkles } from 'lucide-react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { RunAnalysisButton } from '../components/analysis/RunAnalysisButton'
import { AnomalyStatusBadge, AnomalyTypeBadge, SeverityBadge } from '../components/ui/Badges'
import { ConfidenceBar, EmptyState, ErrorState, Skeleton } from '../components/ui/Feedback'
import { GlassCard } from '../components/ui/GlassCard'
import { api } from '../lib/api'
import { useAnalysis } from '../lib/analysis'
import { formatDateTime, formatPct, formatRelative } from '../lib/format'
import { anomalyActionLabel, confidenceLevel, providerLabel, severityLabel } from '../lib/labels'
import type { AnomalyType, Severity } from '../lib/types'
import { useApi } from '../lib/useApi'

const severities: (Severity | '')[] = ['', 'HIGH', 'MEDIUM', 'LOW']
const types: { value: AnomalyType | ''; label: string }[] = [
  { value: '', label: 'Todos los tipos' },
  { value: 'REAL_ANOMALY', label: 'Anomalía real' },
  { value: 'DATA_QUALITY', label: 'Calidad de datos' },
  { value: 'EXPLAINABLE_ANOMALY', label: 'Explicable' },
  { value: 'FALSE_POSITIVE', label: 'Falso positivo' },
]

export function AnomaliesPage() {
  const [params, setParams] = useSearchParams()
  const navigate = useNavigate()
  const { completedVersion, run, running } = useAnalysis()
  const severity = params.get('severity') ?? ''
  const type = params.get('type') ?? ''
  const { data, error, loading, reload } = useApi(() => api.anomalies({ severity, type }), [severity, type, completedVersion])

  const set = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-shadow-glass sm:text-3xl">Anomalías IA</h1>
          <p className="mt-1 text-sm text-ink-soft/80">Hallazgos clasificados, explicados y ordenados por prioridad de investigación.</p>
        </div>
      </div>

      {run && run.status === 'COMPLETED' && (
        <GlassCard className="flex flex-col gap-3 bg-gradient-to-r from-[#06B6D4]/15 via-[#8B5CF6]/10 to-transparent p-5 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <span className="flex size-10 items-center justify-center rounded-xl bg-white/10">
              <Sparkles className="size-5 text-violet-200" aria-hidden />
            </span>
            <div>
              <p className="text-lg font-bold text-white">{run.summary}</p>
              <p className="text-xs text-ink-soft">
                Ejecución #{run.id} · {formatRelative(run.finished_at)} · {run.meters_analyzed} medidores · {providerLabel(run.ai_provider)}
              </p>
            </div>
          </div>
        </GlassCard>
      )}

      <GlassCard dense className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap gap-2" role="group" aria-label="Filtrar por severidad">
          {severities.map((s) => (
            <button
              key={s || 'all'}
              onClick={() => set('severity', s)}
              aria-pressed={severity === s}
              className={clsx(
                'rounded-xl px-3.5 py-2 text-sm font-semibold transition focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]',
                severity === s ? 'bg-white/[0.14] text-white shadow-[inset_0_0_0_1px_rgba(255,255,255,0.2)]' : 'text-ink-soft hover:bg-white/[0.06]',
              )}
            >
              {s ? `Severidad ${severityLabel[s].toLowerCase()}` : 'Todas'}
            </button>
          ))}
        </div>
        <label>
          <span className="sr-only">Tipo de anomalía</span>
          <select
            value={type}
            onChange={(e) => set('type', e.target.value)}
            className="h-10 rounded-xl border border-white/20 bg-[#0F172A] px-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-[#06B6D4]"
          >
            {types.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </label>
      </GlassCard>

      {error ? (
        <ErrorState message={error.message} onRetry={reload} />
      ) : (
        <GlassCard dense className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[900px] text-left text-sm">
              <thead className="border-b border-white/10 text-xs uppercase tracking-wide text-ink-muted">
                <tr>
                  <th scope="col" className="px-5 py-3 font-semibold">#</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Medidor</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Tipo</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Severidad</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Confianza</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Acción</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Estado</th>
                  <th scope="col" className="w-10 px-5 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.06]">
                {loading && !data
                  ? Array.from({ length: 4 }).map((_, i) => (
                      <tr key={i}>
                        <td colSpan={8} className="px-5 py-3">
                          <Skeleton className="h-10" />
                        </td>
                      </tr>
                    ))
                  : data?.items.map((a) => (
                      <tr
                        key={a.id}
                        tabIndex={0}
                        onClick={() => navigate(`/anomalies/${a.id}`)}
                        onKeyDown={(e) => e.key === 'Enter' && navigate(`/anomalies/${a.id}`)}
                        className={clsx('cursor-pointer transition hover:bg-white/[0.05] focus:bg-white/[0.06] focus:outline-none', a.type === 'FALSE_POSITIVE' && 'opacity-75')}
                      >
                        <td className="px-5 py-4">
                          <span className="tabular flex size-7 items-center justify-center rounded-lg bg-white/10 text-xs font-bold">{a.priority}</span>
                        </td>
                        <td className="px-5 py-4">
                          <p className="font-semibold text-white">{a.meter_id}</p>
                          <p className="text-xs text-ink-soft/80">
                            {formatPct(a.variation_pct)} vs baseline · {formatDateTime(a.detected_at)}
                          </p>
                        </td>
                        <td className="px-5 py-4">
                          <AnomalyTypeBadge type={a.type} />
                        </td>
                        <td className="px-5 py-4">
                          <SeverityBadge severity={a.severity} />
                        </td>
                        <td className="px-5 py-4">
                          <div className="flex items-center gap-2">
                            <ConfidenceBar value={a.confidence} />
                            <span className="text-xs text-ink-soft">{confidenceLevel(a.confidence)}</span>
                          </div>
                        </td>
                        <td className="px-5 py-4 font-semibold text-white">{anomalyActionLabel[a.type]}</td>
                        <td className="px-5 py-4">
                          <AnomalyStatusBadge status={a.status} />
                        </td>
                        <td className="px-5 py-4 text-ink-muted">
                          <ChevronRight className="size-4" aria-hidden />
                        </td>
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>
          {data && data.items.length === 0 && (
            <EmptyState
              icon={<BrainCircuit className="size-6" aria-hidden />}
              title={run ? 'Sin anomalías para estos filtros' : 'Aún no se ha ejecutado el análisis'}
              description={run ? 'Prueba con otros filtros o vuelve a ejecutar el análisis.' : 'Run AI Analysis recorre lecturas, baseline, detección, correlación y eventos para explicar y priorizar anomalías.'}
              action={!run && !running && <RunAnalysisButton size="sm" />}
            />
          )}
        </GlassCard>
      )}
    </div>
  )
}
