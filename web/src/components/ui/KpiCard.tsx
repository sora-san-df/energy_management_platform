import clsx from 'clsx'
import type { ReactNode } from 'react'
import { GlassCard } from './GlassCard'

interface KpiCardProps {
  label: string
  value: ReactNode
  hint?: ReactNode
  icon: ReactNode
  accent?: 'cyan' | 'purple' | 'critical' | 'warning' | 'good'
}

const accents = {
  cyan: 'from-[#06B6D4]/30 text-cyan-200',
  purple: 'from-[#8B5CF6]/30 text-violet-200',
  critical: 'from-red-500/30 text-red-200',
  warning: 'from-amber-500/30 text-amber-100',
  good: 'from-emerald-500/30 text-emerald-100',
}

export function KpiCard({ label, value, hint, icon, accent = 'cyan' }: KpiCardProps) {
  return (
    <GlassCard className="relative overflow-hidden p-5">
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm font-medium text-ink-soft">{label}</p>
        <span className={clsx('flex size-9 items-center justify-center rounded-xl bg-gradient-to-br to-transparent', accents[accent])}>{icon}</span>
      </div>
      <div className="tabular mt-3 text-3xl font-bold tracking-tight text-white text-shadow-glass">{value}</div>
      {hint && <div className="mt-1.5 text-xs text-ink-soft/80">{hint}</div>}
    </GlassCard>
  )
}
