import { Zap } from 'lucide-react'

export function Logo({ compact = false }: { compact?: boolean }) {
  return (
    <div className="flex items-center gap-2.5">
      <span className="flex size-9 items-center justify-center rounded-xl border border-white/20 bg-gradient-to-br from-[#06B6D4]/40 to-[#8B5CF6]/40 shadow-[0_0_24px_-4px_rgba(6,182,212,0.6)]">
        <Zap className="size-5 text-white" aria-hidden />
      </span>
      {!compact && (
        <div className="leading-tight">
          <p className="text-base font-bold tracking-tight text-white">Voltia</p>
          <p className="text-[11px] font-medium uppercase tracking-[0.14em] text-ink-muted">Energy Intelligence</p>
        </div>
      )}
    </div>
  )
}
