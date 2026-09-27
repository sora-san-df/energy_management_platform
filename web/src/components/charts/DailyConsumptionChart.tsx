import { Area, CartesianGrid, ComposedChart, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { formatDate, formatKWh, formatNumber, formatPct } from '../../lib/format'
import type { DailyPoint } from '../../lib/types'
import { ChartTooltip, Legend } from './chartTheme'
import { axisProps, chart } from './chartTokens'

interface DailyConsumptionChartProps {
  data: DailyPoint[]
  height?: number
}

export function DailyConsumptionChart({ data, height = 280 }: DailyConsumptionChartProps) {
  const byDate = new Map(data.map((d) => [d.date, d]))
  return (
    <div>
      <Legend
        items={[
          { label: 'Consumo diario', color: chart.series1 },
          { label: 'Baseline (mediana)', color: chart.series2, dashed: true },
        ]}
      />
      <div className="mt-3" style={{ height }} role="img" aria-label="Consumo diario comparado con el baseline">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="dailyFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={chart.series1} stopOpacity={0.35} />
                <stop offset="100%" stopColor={chart.series1} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} stroke={chart.grid} />
            <XAxis dataKey="date" tickFormatter={formatDate} {...axisProps} minTickGap={24} />
            <YAxis {...axisProps} width={64} tickFormatter={(v: number) => formatNumber(v)} />
            <Tooltip
              cursor={{ stroke: 'rgba(255,255,255,0.25)', strokeWidth: 1 }}
              content={(props) => (
                <ChartTooltip
                  {...(props as object)}
                  labelFormatter={formatDate}
                  valueFormatter={(v) => formatKWh(v)}
                  footer={(label) => {
                    const p = byDate.get(label)
                    if (!p || !p.baseline_kwh) return null
                    const diff = ((p.kwh - p.baseline_kwh) / p.baseline_kwh) * 100
                    return <p className="mt-2 border-t border-white/10 pt-2 text-ink-soft">Variación: <span className="font-semibold text-white">{formatPct(diff)}</span></p>
                  }}
                />
              )}
            />
            <Area type="monotone" dataKey="kwh" name="Consumo diario" stroke={chart.series1} strokeWidth={2} fill="url(#dailyFill)" activeDot={{ r: 5, stroke: chart.surface, strokeWidth: 2 }} />
            <Line type="monotone" dataKey="baseline_kwh" name="Baseline" stroke={chart.series2} strokeWidth={2} strokeDasharray="6 4" dot={false} activeDot={false} />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
