import { CartesianGrid, Line, LineChart, ReferenceArea, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { formatDate, formatDateTime, formatNumber } from '../../lib/format'
import type { SeriesPoint } from '../../lib/types'
import { ChartTooltip } from './chartTheme'
import { axisProps, chart } from './chartTokens'

type Metric = 'voltage_v' | 'current_a' | 'power_factor'

const metrics: { key: Metric; title: string; unit: string; decimals: number; domain?: [number, number]; band?: [number, number] }[] = [
  { key: 'voltage_v', title: 'Voltaje', unit: 'V', decimals: 1, domain: [195, 245], band: [209, 231] },
  { key: 'current_a', title: 'Corriente', unit: 'A', decimals: 0 },
  { key: 'power_factor', title: 'Factor de potencia', unit: '', decimals: 2, domain: [0.5, 1] },
]

export function ElectricalCharts({ points, highlight }: { points: SeriesPoint[]; highlight?: string[] }) {
  return (
    <div className="grid gap-4 md:grid-cols-3">
      {metrics.map((m) => {
        const affected = highlight?.includes(m.key)
        return (
          <div key={m.key} className="rounded-xl border border-white/10 bg-white/[0.03] p-3">
            <div className="mb-2 flex items-center justify-between">
              <p className="text-sm font-semibold text-white">{m.title}</p>
              {affected && <span className="rounded-full bg-status-critical/15 px-2 py-0.5 text-[11px] font-semibold text-red-200">Variable afectada</span>}
            </div>
            <div className="h-36" role="img" aria-label={`Serie de ${m.title}`}>
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={points} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
                  <CartesianGrid vertical={false} stroke={chart.grid} />
                  <XAxis dataKey="timestamp" tickFormatter={formatDate} {...axisProps} minTickGap={40} />
                  <YAxis {...axisProps} width={40} domain={m.domain ?? ['auto', 'auto']} tickFormatter={(v: number) => formatNumber(v, m.key === 'power_factor' ? 2 : 0)} />
                  {m.band && <ReferenceArea y1={m.band[0]} y2={m.band[1]} fill="#34d399" fillOpacity={0.06} />}
                  <Tooltip
                    cursor={{ stroke: 'rgba(255,255,255,0.25)', strokeWidth: 1 }}
                    content={(props) => (
                      <ChartTooltip {...(props as object)} labelFormatter={formatDateTime} valueFormatter={(v) => `${formatNumber(v, m.decimals)} ${m.unit}`.trim()} />
                    )}
                  />
                  <Line type="linear" dataKey={m.key} name={m.title} stroke={chart.series1} strokeWidth={1.5} dot={false} activeDot={{ r: 4, stroke: chart.surface, strokeWidth: 2 }} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>
        )
      })}
    </div>
  )
}
