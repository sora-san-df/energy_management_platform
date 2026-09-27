const numberFormatters = new Map<number, Intl.NumberFormat>()

function nf(decimals: number) {
  let f = numberFormatters.get(decimals)
  if (!f) {
    f = new Intl.NumberFormat('es-CO', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
    numberFormatters.set(decimals, f)
  }
  return f
}

export function formatNumber(value: number | undefined | null, decimals = 0) {
  if (value === undefined || value === null || Number.isNaN(value)) return '—'
  return nf(decimals).format(value)
}

export function formatKWh(value: number | undefined | null, decimals = 0) {
  return `${formatNumber(value, decimals)} kWh`
}

export function formatCompactKWh(value: number) {
  if (Math.abs(value) >= 1000) return `${formatNumber(value / 1000, 1)} MWh`
  return formatKWh(value)
}

export function formatPct(value: number | undefined | null, decimals = 1) {
  if (value === undefined || value === null) return '—'
  const sign = value > 0 ? '+' : ''
  return `${sign}${formatNumber(value, decimals)}%`
}

export function formatConfidence(value: number) {
  return `${Math.round(value * 100)}%`
}

const dateTime = new Intl.DateTimeFormat('es-CO', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'UTC' })
const dateOnly = new Intl.DateTimeFormat('es-CO', { day: '2-digit', month: 'short', timeZone: 'UTC' })
const localDateTime = new Intl.DateTimeFormat('es-CO', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit', hour12: false })

export function formatDateTime(value?: string) {
  return value ? dateTime.format(new Date(value)) : '—'
}

export function formatDate(value?: string) {
  return value ? dateOnly.format(new Date(value)) : '—'
}

export function formatLocalDateTime(value?: string) {
  return value ? localDateTime.format(new Date(value)) : '—'
}

export function formatRelative(value?: string) {
  if (!value) return 'Nunca'
  const diff = Date.now() - new Date(value).getTime()
  const minutes = Math.round(diff / 60000)
  if (minutes < 1) return 'Hace instantes'
  if (minutes < 60) return `Hace ${minutes} min`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `Hace ${hours} h`
  return formatLocalDateTime(value)
}

export function formatDuration(ms: number) {
  if (ms < 1000) return `${ms} ms`
  return `${formatNumber(ms / 1000, 1)} s`
}
