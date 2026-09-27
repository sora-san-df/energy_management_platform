import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from './api'

export interface ApiState<T> {
  data: T | undefined
  error: ApiError | undefined
  loading: boolean
  reload: () => void
}

interface Result<T> {
  key: string
  data?: T
  error?: ApiError
}

export function useApi<T>(fetcher: () => Promise<T>, deps: unknown[]): ApiState<T> {
  const [nonce, setNonce] = useState(0)
  const key = `${JSON.stringify(deps)}#${nonce}`
  const [result, setResult] = useState<Result<T>>({ key: '' })
  const fetcherRef = useRef(fetcher)

  useEffect(() => {
    fetcherRef.current = fetcher
  })

  useEffect(() => {
    let active = true
    fetcherRef
      .current()
      .then((data) => {
        if (active) setResult({ key, data })
      })
      .catch((err: unknown) => {
        if (!active) return
        const error = err instanceof ApiError ? err : new ApiError(0, 'unknown', 'Error inesperado')
        setResult((prev) => ({ key, data: prev.data, error }))
      })
    return () => {
      active = false
    }
  }, [key])

  const reload = useCallback(() => setNonce((n) => n + 1), [])
  return { data: result.data, error: result.error, loading: result.key !== key, reload }
}
