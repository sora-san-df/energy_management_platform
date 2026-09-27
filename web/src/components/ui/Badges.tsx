import clsx from 'clsx'
import { AlertOctagon, AlertTriangle, CheckCircle2, CircleDashed, Database, FileSearch, Info, ShieldCheck, Sparkles, Workflow } from 'lucide-react'
import type { ReactNode } from 'react'
import { anomalyStatusLabel, anomalyTypeLabel, meterStatusLabel, severityLabel } from '../../lib/labels'
import type { AnomalyStatus, AnomalyType, MeterStatus, Severity } from '../../lib/types'

type Tone = 'critical' | 'warning' | 'good' | 'info' | 'neutral' | 'violet'

const tones: Record<Tone, string> = {
  critical: 'border-status-critical/40 bg-status-critical/12 text-red-200',
  warning: 'border-status-warning/40 bg-status-warning/12 text-amber-100',
  good: 'border-status-good/40 bg-status-good/12 text-emerald-100',
  info: 'border-status-info/40 bg-status-info/12 text-sky-100',
  neutral: 'border-white/20 bg-white/[0.06] text-ink-soft',
  violet: 'border-aurora-purple/40 bg-aurora-purple/15 text-violet-100',
}

const iconTones: Record<Tone, string> = {
  critical: 'text-status-critical',
  warning: 'text-status-warning',
  good: 'text-status-good',
  info: 'text-status-info',
  neutral: 'text-ink-muted',
  violet: 'text-violet-300',
}

export function Pill({ tone, icon, children, className }: { tone: Tone; icon?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <span className={clsx('inline-flex items-center gap-1.5 whitespace-nowrap rounded-full border px-2.5 py-0.5 text-xs font-semibold', tones[tone], className)}>
      {icon && <span className={clsx('flex shrink-0', iconTones[tone])}>{icon}</span>}
      {children}
    </span>
  )
}

const severityTone: Record<Severity, Tone> = { HIGH: 'critical', MEDIUM: 'warning', LOW: 'neutral' }

export function SeverityBadge({ severity }: { severity: Severity }) {
  const Icon = severity === 'HIGH' ? AlertOctagon : severity === 'MEDIUM' ? AlertTriangle : Info
  return (
    <Pill tone={severityTone[severity]} icon={<Icon className="size-3.5" aria-hidden />}>
      {severityLabel[severity]}
    </Pill>
  )
}

const typeTone: Record<AnomalyType, Tone> = {
  REAL_ANOMALY: 'critical',
  DATA_QUALITY: 'info',
  EXPLAINABLE_ANOMALY: 'violet',
  FALSE_POSITIVE: 'neutral',
}

const typeIcon: Record<AnomalyType, typeof AlertOctagon> = {
  REAL_ANOMALY: AlertOctagon,
  DATA_QUALITY: Database,
  EXPLAINABLE_ANOMALY: Workflow,
  FALSE_POSITIVE: ShieldCheck,
}

export function AnomalyTypeBadge({ type }: { type: AnomalyType }) {
  const Icon = typeIcon[type]
  return (
    <Pill tone={typeTone[type]} icon={<Icon className="size-3.5" aria-hidden />}>
      {anomalyTypeLabel[type]}
    </Pill>
  )
}

const meterTone: Record<MeterStatus, Tone> = { OK: 'good', ALERT: 'warning', CRITICAL: 'critical', PENDING: 'neutral' }
const meterIcon: Record<MeterStatus, typeof AlertOctagon> = { OK: CheckCircle2, ALERT: AlertTriangle, CRITICAL: AlertOctagon, PENDING: CircleDashed }

export function MeterStatusBadge({ status }: { status: MeterStatus }) {
  const Icon = meterIcon[status]
  return (
    <Pill tone={meterTone[status]} icon={<Icon className="size-3.5" aria-hidden />}>
      {meterStatusLabel[status]}
    </Pill>
  )
}

const statusTone: Record<AnomalyStatus, Tone> = { OPEN: 'warning', INVESTIGATING: 'info', RESOLVED: 'good', DISMISSED: 'neutral' }

export function AnomalyStatusBadge({ status }: { status: AnomalyStatus }) {
  return <Pill tone={statusTone[status]}>{anomalyStatusLabel[status]}</Pill>
}

export function SourceBadge({ source }: { source: 'gemini' | 'rules' }) {
  return source === 'gemini' ? (
    <Pill tone="violet" icon={<Sparkles className="size-3.5" aria-hidden />}>
      Explicado por Gemini
    </Pill>
  ) : (
    <Pill tone="neutral" icon={<FileSearch className="size-3.5" aria-hidden />}>
      Motor de reglas
    </Pill>
  )
}
