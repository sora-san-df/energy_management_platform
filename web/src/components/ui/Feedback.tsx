import clsx from 'clsx'
import { AlertCircle, RefreshCw } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from './Button'
import { GlassCard } from './GlassCard'

export function Skeleton({ className }: { className?: string }) {
  return <div className={clsx('skeleton rounded-lg', className)} aria-hidden />
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <GlassCard dense className="flex flex-col items-center gap-3 px-6 py-10 text-center" role="alert">
      <AlertCircle className="size-8 text-status-critical" aria-hidden />
      <p className="text-sm text-ink-soft">{message}</p>
      {onRetry && (
        <Button size="sm" onClick={onRetry} icon={<RefreshCw className="size-3.5" aria-hidden />}>
          Reintentar
        </Button>
      )}
    </GlassCard>
  )
}

export function EmptyState({ icon, title, description, action }: { icon: ReactNode; title: string; description: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-3 px-6 py-10 text-center">
      <div className="flex size-12 items-center justify-center rounded-2xl border border-white/15 bg-white/[0.06] text-aurora-cyan">{icon}</div>
      <div>
        <p className="font-semibold text-white">{title}</p>
        <p className="mt-1 max-w-md text-sm text-ink-soft/80">{description}</p>
      </div>
      {action}
    </div>
  )
}

export function ConfidenceBar({ value, className }: { value: number; className?: string }) {
  const pct = Math.round(value * 100)
  return (
    <div className={clsx('flex items-center gap-2', className)}>
      <div className="h-1.5 w-16 overflow-hidden rounded-full bg-white/10" role="meter" aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct} aria-label="Confianza">
        <div className="h-full rounded-full bg-gradient-to-r from-[#06B6D4] to-[#8B5CF6]" style={{ width: `${pct}%` }} />
      </div>
      <span className="tabular text-xs font-semibold text-white">{pct}%</span>
    </div>
  )
}
