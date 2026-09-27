import clsx from 'clsx'
import { AlertTriangle, CheckCircle2, Circle, Loader2, Sparkles, X, XCircle } from 'lucide-react'
import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useAnalysis } from '../../lib/analysis'
import { formatConfidence, formatDuration } from '../../lib/format'
import type { AIStatus, StepStatus } from '../../lib/types'
import { Pill } from '../ui/Badges'
import { Button } from '../ui/Button'

const stepIcon: Record<StepStatus, React.ReactNode> = {
  PENDING: <Circle className="size-5 text-ink-muted/60" aria-hidden />,
  RUNNING: <Loader2 className="size-5 animate-spin text-aurora-cyan" aria-hidden />,
  DONE: <CheckCircle2 className="size-5 text-status-good" aria-hidden />,
  FAILED: <XCircle className="size-5 text-status-critical" aria-hidden />,
}

const aiStatusCopy: Record<Exclude<AIStatus, ''>, { tone: 'violet' | 'warning' | 'neutral'; text: string }> = {
  ok: { tone: 'violet', text: 'Explicaciones generadas con Gemini' },
  partial: { tone: 'warning', text: 'Gemini parcial · fallback a reglas en algunos casos' },
  fallback: { tone: 'warning', text: 'Gemini no respondió · explicaciones del motor de reglas' },
  disabled: { tone: 'neutral', text: 'Gemini no configurado · explicaciones del motor de reglas' },
}

export function AnalysisPanel() {
  const { run, running, error, panelOpen, closePanel } = useAnalysis()

  useEffect(() => {
    if (!panelOpen) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && closePanel()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [panelOpen, closePanel])

  if (!panelOpen) return null
  const done = run?.steps.filter((s) => s.status === 'DONE').length ?? 0
  const total = run?.steps.length ?? 7
  const ai = run?.ai_status ? aiStatusCopy[run.ai_status] : undefined

  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true" aria-labelledby="analysis-title">
      <button className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={closePanel} aria-label="Cerrar panel" />
      <aside className="relative flex h-full w-full max-w-md flex-col border-l border-white/20 bg-[#0F172A]/85 shadow-[0_8px_32px_0_rgba(0,0,0,0.37)] backdrop-blur-[24px]">
        <header className="flex items-start justify-between gap-3 border-b border-white/10 p-5">
          <div>
            <div className="flex items-center gap-2 text-aurora-cyan">
              <Sparkles className="size-4" aria-hidden />
              <span className="text-xs font-semibold uppercase tracking-[0.14em]">Análisis IA</span>
            </div>
            <h2 id="analysis-title" className="mt-1 text-lg font-bold text-white">
              {run ? `Ejecución #${run.id}` : 'Iniciando análisis'}
            </h2>
            <p className="text-sm text-ink-soft/80">
              {run?.scope.length ? `Medidores: ${run.scope.join(', ')}` : 'Todos los medidores'}
            </p>
          </div>
          <Button variant="ghost" size="sm" onClick={closePanel} aria-label="Cerrar">
            <X className="size-4" aria-hidden />
          </Button>
        </header>

        <div className="flex-1 overflow-y-auto p-5">
          {error && (
            <div className="mb-4 flex gap-2 rounded-xl border border-status-critical/40 bg-status-critical/10 p-3 text-sm text-red-100" role="alert">
              <AlertTriangle className="size-4 shrink-0 text-status-critical" aria-hidden />
              {error}
            </div>
          )}

          <div className="mb-5">
            <div className="mb-2 flex justify-between text-xs text-ink-soft">
              <span>Lecturas → Baseline → Detección → Correlación → Eventos → Explicación → Recomendación</span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-white/10">
              <div className="h-full rounded-full bg-gradient-to-r from-[#06B6D4] to-[#8B5CF6] transition-all duration-500" style={{ width: `${(done / total) * 100}%` }} />
            </div>
          </div>

          <ol className="space-y-1" aria-live="polite">
            {(run?.steps ?? []).map((step, i) => (
              <li key={step.key} className={clsx('flex gap-3 rounded-xl p-3 transition', step.status === 'RUNNING' && 'bg-white/[0.06]')}>
                <div className="flex flex-col items-center">
                  {stepIcon[step.status]}
                  {i < total - 1 && <span className="mt-1 w-px flex-1 bg-white/10" />}
                </div>
                <div className="min-w-0 pb-1">
                  <p className={clsx('text-sm font-semibold', step.status === 'PENDING' ? 'text-ink-muted' : 'text-white')}>{step.label}</p>
                  {step.detail && <p className="mt-0.5 text-xs text-ink-soft/80">{step.detail}</p>}
                </div>
              </li>
            ))}
          </ol>

          {run && run.status !== 'RUNNING' && (
            <div className={clsx('mt-5 rounded-2xl border p-4', run.status === 'COMPLETED' ? 'border-aurora-cyan/30 bg-aurora-cyan/10' : 'border-status-critical/40 bg-status-critical/10')}>
              <p className="text-base font-bold text-white">{run.summary}</p>
              {run.status === 'COMPLETED' ? (
                <dl className="mt-3 grid grid-cols-3 gap-3 text-center">
                  <div>
                    <dt className="text-[11px] uppercase tracking-wide text-ink-muted">Medidores</dt>
                    <dd className="tabular text-lg font-bold text-white">{run.meters_analyzed}</dd>
                  </div>
                  <div>
                    <dt className="text-[11px] uppercase tracking-wide text-ink-muted">Confianza</dt>
                    <dd className="tabular text-lg font-bold text-white">{run.anomalies_detected ? formatConfidence(run.avg_confidence) : '—'}</dd>
                  </div>
                  <div>
                    <dt className="text-[11px] uppercase tracking-wide text-ink-muted">Duración</dt>
                    <dd className="tabular text-lg font-bold text-white">{formatDuration(run.duration_ms)}</dd>
                  </div>
                </dl>
              ) : (
                <p className="mt-1 text-sm text-red-100">{run.error}</p>
              )}
              {ai && (
                <Pill tone={ai.tone} className="mt-3" icon={<Sparkles className="size-3.5" aria-hidden />}>
                  {ai.text}
                </Pill>
              )}
            </div>
          )}
        </div>

        <footer className="border-t border-white/10 p-5">
          {running ? (
            <p className="text-center text-sm text-ink-soft">Puedes seguir navegando; te avisaremos al terminar.</p>
          ) : (
            <Link
              to="/anomalies"
              onClick={closePanel}
              className="flex h-10 w-full items-center justify-center rounded-xl bg-gradient-to-r from-[#06B6D4] to-[#8B5CF6] text-sm font-semibold text-white hover:brightness-110 focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]"
            >
              Ver anomalías priorizadas
            </Link>
          )}
        </footer>
      </aside>
    </div>
  )
}
