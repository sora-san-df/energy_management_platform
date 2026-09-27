import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { ApiError, api } from './api'
import type { AnalysisRun } from './types'

interface AnalysisContextValue {
  run: AnalysisRun | null
  running: boolean
  error: string | null
  panelOpen: boolean
  completedVersion: number
  start: (meterIds?: string[]) => Promise<void>
  openPanel: () => void
  closePanel: () => void
}

const AnalysisContext = createContext<AnalysisContextValue | null>(null)

const POLL_MS = 600

export function AnalysisProvider({ children }: { children: ReactNode }) {
  const [run, setRun] = useState<AnalysisRun | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [panelOpen, setPanelOpen] = useState(false)
  const [completedVersion, setCompletedVersion] = useState(0)

  useEffect(() => {
    api.latestAnalysis().then(setRun).catch(() => undefined)
  }, [])

  useEffect(() => {
    if (!run || run.status !== 'RUNNING') return
    const timer = window.setTimeout(async () => {
      try {
        const next = await api.analysis(run.id)
        setRun(next)
        if (next.status !== 'RUNNING') setCompletedVersion((v) => v + 1)
      } catch {
        setRun((current) => (current ? { ...current } : current))
      }
    }, POLL_MS)
    return () => window.clearTimeout(timer)
  }, [run])

  const start = useCallback(async (meterIds?: string[]) => {
    setError(null)
    setPanelOpen(true)
    try {
      setRun(await api.analyze(meterIds))
    } catch (err) {
      const current = err instanceof ApiError && err.status === 409 ? (err.body as { analysis?: AnalysisRun })?.analysis : undefined
      if (current) {
        setRun(current)
        return
      }
      setError(err instanceof ApiError ? err.message : 'No se pudo iniciar el análisis')
    }
  }, [])

  const openPanel = useCallback(() => setPanelOpen(true), [])
  const closePanel = useCallback(() => setPanelOpen(false), [])

  const value = useMemo<AnalysisContextValue>(
    () => ({ run, running: run?.status === 'RUNNING', error, panelOpen, completedVersion, start, openPanel, closePanel }),
    [run, error, panelOpen, completedVersion, start, openPanel, closePanel],
  )
  return <AnalysisContext.Provider value={value}>{children}</AnalysisContext.Provider>
}

export function useAnalysis() {
  const ctx = useContext(AnalysisContext)
  if (!ctx) throw new Error('useAnalysis must be used inside AnalysisProvider')
  return ctx
}
