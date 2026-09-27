export const chart = {
  series1: '#0891b2',
  series2: '#8b5cf6',
  series3: '#d97706',
  critical: '#f87171',
  grid: 'rgba(255,255,255,0.08)',
  axis: '#94a3b8',
  surface: '#131c31',
}

export const axisProps = {
  stroke: chart.grid,
  tick: { fill: chart.axis, fontSize: 11 },
  tickLine: false,
  axisLine: { stroke: chart.grid },
}
