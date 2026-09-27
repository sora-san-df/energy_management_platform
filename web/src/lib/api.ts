import type {
  AnalysisRun,
  Anomaly,
  AnomalyStatus,
  DashboardSummary,
  EnergyEvent,
  ListResponse,
  MeterDetail,
  MeterOverview,
  ReadingsResult,
  Session,
  User,
} from './types'

const TOKEN_KEY = 'voltia.session'

export class ApiError extends Error {
  status: number
  code: string
  body: unknown

  constructor(status: number, code: string, message: string, body?: unknown) {
    super(message)
    this.status = status
    this.code = code
    this.body = body
  }
}

export function loadSession(): Session | null {
  try {
    const raw = localStorage.getItem(TOKEN_KEY)
    if (!raw) return null
    const session = JSON.parse(raw) as Session
    if (new Date(session.expires_at).getTime() <= Date.now()) {
      localStorage.removeItem(TOKEN_KEY)
      return null
    }
    return session
  } catch {
    return null
  }
}

export function saveSession(session: Session | null) {
  try {
    if (session) localStorage.setItem(TOKEN_KEY, JSON.stringify(session))
    else localStorage.removeItem(TOKEN_KEY)
  } catch {
    return
  }
}

let unauthorizedHandler: (() => void) | null = null

export function onUnauthorized(handler: (() => void) | null) {
  unauthorizedHandler = handler
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body) headers.set('Content-Type', 'application/json')
  const session = loadSession()
  if (session) headers.set('Authorization', `Bearer ${session.access_token}`)

  let res: Response
  try {
    res = await fetch(`/api${path}`, { ...init, headers })
  } catch {
    throw new ApiError(0, 'network_error', 'No se pudo conectar con el servidor')
  }

  const text = await res.text()
  const body = text ? JSON.parse(text) : undefined
  if (!res.ok) {
    if (res.status === 401 && !path.startsWith('/auth/login')) unauthorizedHandler?.()
    const err = body?.error ?? {}
    throw new ApiError(res.status, err.code ?? 'http_error', err.message ?? `Error ${res.status}`, body)
  }
  return body as T
}

export interface MeterQuery {
  status?: string
  search?: string
  sort?: string
  order?: string
}

function qs(params: Record<string, string | undefined>) {
  const search = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => {
    if (v) search.set(k, v)
  })
  const s = search.toString()
  return s ? `?${s}` : ''
}

export const api = {
  login: (email: string, password: string) =>
    request<Session>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  me: () => request<User>('/auth/me'),
  dashboard: () => request<DashboardSummary>('/dashboard/summary'),
  meters: (q: MeterQuery = {}) => request<ListResponse<MeterOverview>>(`/meters${qs({ ...q })}`),
  meter: (id: string) => request<MeterDetail>(`/meters/${encodeURIComponent(id)}`),
  readings: (id: string, granularity: 'hour' | 'day' = 'hour') =>
    request<ReadingsResult>(`/meters/${encodeURIComponent(id)}/readings${qs({ granularity })}`),
  events: (meterId?: string) => request<ListResponse<EnergyEvent>>(`/events${qs({ meter_id: meterId })}`),
  anomalies: (q: { severity?: string; type?: string; status?: string; meter_id?: string } = {}) =>
    request<ListResponse<Anomaly>>(`/anomalies${qs(q)}`),
  anomaly: (id: number | string) => request<Anomaly>(`/anomalies/${id}`),
  updateAnomalyStatus: (id: number, status: AnomalyStatus) =>
    request<Anomaly>(`/anomalies/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) }),
  analyze: (meterIds?: string[]) =>
    request<AnalysisRun>('/ai/analyze', { method: 'POST', body: JSON.stringify(meterIds?.length ? { meter_ids: meterIds } : {}) }),
  analysis: (id: number) => request<AnalysisRun>(`/ai/analysis/${id}`),
  latestAnalysis: () => request<AnalysisRun>('/ai/analysis/latest'),
}
