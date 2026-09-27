import { CartesianGrid, ComposedChart, Line, ReferenceArea, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { formatDate, formatDateTime, formatNumber, formatPct } from '../../lib/format'
import { eventTypeLabel } from '../../lib/labels'
import type { EnergyEvent, SeriesPoint } from '../../lib/types'
import { ChartTooltip, Legend } from './chartTheme'
import { axisProps, chart } from './chartTokens'

interface HourlyConsumptionChartProps {
  points: SeriesPoint[]
  window?: { start?: string; end?: string; label: string }
  events?: EnergyEvent[]
  height?: number
}

function snap(points: SeriesPoint[], ts?: string) {
  if (!ts || points.length === 0) return undefined
  const target = new Date(ts).getTime()
  let best = points[0].timestamp
  for (const p of points) {
    if (new Date(p.timestamp).getTime() <= target) best = p.timestamp
  }
  return best
}

export function HourlyConsumptionChart({ points, window, events = [], height = 300 }: HourlyConsumptionChartProps) {
  const byTs = new Map(points.map((p) => [p.timestamp, p]))
  const windowStart = snap(points, window?.start)
  const windowEnd = window?.end ? snap(points, new Date(new Date(window.end).getTime() - 3600_000).toISOString()) : undefined
  const eventMarks = events.map((e) => ({ ...e, x: snap(points, e.timestamp) })).filter((e) => e.x)

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Legend
          items={[
            { label: 'Consumo real (kWh/h)', color: chart.series1 },
            { label: 'Esperado por hora (baseline)', color: chart.series2, dashed: true },
          ]}
        />
        {window && windowStart && (
          <span className="flex items-center gap-2 text-xs text-ink-soft">
            <span className="h-3 w-4 rounded-sm border border-status-critical/60 bg-status-critical/20" aria-hidden />
            {window.label}
          </span>
        )}
      </div>
      <div className="mt-3" style={{ height }} role="img" aria-label="Consumo horario comparado con el perfil esperado">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={points} margin={{ top: 16, right: 8, bottom: 0, left: 0 }}>
            <CartesianGrid vertical={false} stroke={chart.grid} />
            <XAxis dataKey="timestamp" tickFormatter={formatDate} {...axisProps} minTickGap={48} />
            <YAxis {...axisProps} width={48} tickFormatter={(v: number) => formatNumber(v)} />
            {windowStart && windowEnd && (
              <ReferenceArea x1={windowStart} x2={windowEnd} fill={chart.critical} fillOpacity={0.12} stroke={chart.critical} strokeOpacity={0.5} ifOverflow="extendDomain" />
            )}
            {eventMarks.map((e) => (
              <ReferenceLine
                key={e.id}
                x={e.x}
                stroke="#e2e8f0"
                strokeOpacity={0.6}
                label={{ value: eventTypeLabel[e.type] ?? e.type, position: 'insideTopLeft', fill: '#e2e8f0', fontSize: 11 }}
              />
            ))}
            <Tooltip
              cursor={{ stroke: 'rgba(255,255,255,0.25)', strokeWidth: 1 }}
              content={(props) => (
                <ChartTooltip
                  {...(props as object)}
                  labelFormatter={formatDateTime}
                  valueFormatter={(v) => `${formatNumber(v, 1)} kWh`}
                  footer={(label) => {
                    const p = byTs.get(label)
                    if (!p || !p.expected_kwh) return null
                    return (
                      <p className="mt-2 border-t border-white/10 pt-2 text-ink-soft">
                        Desviación: <span className="font-semibold text-white">{formatPct(((p.consumption_kwh - p.expected_kwh) / p.expected_kwh) * 100)}</span>
                      </p>
                    )
                  }}
                />
              )}
            />
            <Line type="monotone" dataKey="expected_kwh" name="Esperado" stroke={chart.series2} strokeWidth={2} strokeDasharray="6 4" dot={false} activeDot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="consumption_kwh" name="Consumo" stroke={chart.series1} strokeWidth={2} dot={false} activeDot={{ r: 5, stroke: chart.surface, strokeWidth: 2 }} isAnimationActive={false} />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
