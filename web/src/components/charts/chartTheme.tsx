import type { ReactNode } from 'react'

interface TooltipRow {
  name?: string | number
  value?: number | string | (number | string)[]
  color?: string
  dataKey?: string | number
}

interface ChartTooltipProps {
  active?: boolean
  label?: string | number
  payload?: TooltipRow[]
  labelFormatter: (label: string) => string
  valueFormatter: (value: number, key: string) => string
  footer?: (label: string) => ReactNode
}

export function ChartTooltip({ active, label, payload, labelFormatter, valueFormatter, footer }: ChartTooltipProps) {
  if (!active || !payload?.length || label === undefined) return null
  return (
    <div className="min-w-44 rounded-xl border border-white/20 bg-[#0F172A]/90 p-3 text-xs shadow-[0_8px_32px_0_rgba(0,0,0,0.37)] backdrop-blur-[24px]">
      <p className="mb-2 font-semibold text-white">{labelFormatter(String(label))}</p>
      <ul className="space-y-1">
        {payload.map((row) => (
          <li key={String(row.dataKey)} className="flex items-center justify-between gap-4">
            <span className="flex items-center gap-2 text-ink-soft">
              <span className="size-2.5 rounded-full" style={{ background: row.color }} aria-hidden />
              {row.name}
            </span>
            <span className="tabular font-semibold text-white">{valueFormatter(Number(row.value), String(row.dataKey))}</span>
          </li>
        ))}
      </ul>
      {footer?.(String(label))}
    </div>
  )
}

export function Legend({ items }: { items: { label: string; color: string; dashed?: boolean }[] }) {
  return (
    <ul className="flex flex-wrap items-center gap-4 text-xs text-ink-soft" aria-label="Leyenda">
      {items.map((item) => (
        <li key={item.label} className="flex items-center gap-2">
          <svg width="18" height="6" aria-hidden>
            <line x1="0" y1="3" x2="18" y2="3" stroke={item.color} strokeWidth="2" strokeDasharray={item.dashed ? '4 3' : undefined} strokeLinecap="round" />
          </svg>
          {item.label}
        </li>
      ))}
    </ul>
  )
}
