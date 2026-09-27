import { BrainCircuit, Gauge, Lock, Mail, ShieldCheck, Sparkles } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { AuroraBackground } from '../components/layout/AuroraBackground'
import { Logo } from '../components/layout/Logo'
import { Button } from '../components/ui/Button'
import { GlassCard } from '../components/ui/GlassCard'
import { ApiError } from '../lib/api'
import { useAuth } from '../lib/auth'

const highlights = [
  { icon: Gauge, title: 'Monitoreo continuo', text: '12 medidores, consumo, voltaje, corriente y factor de potencia.' },
  { icon: BrainCircuit, title: 'Detección explicable', text: 'Baseline, cambios persistentes, outliers y calidad de datos.' },
  { icon: Sparkles, title: 'IA con Gemini', text: 'Explicación, confianza y acción recomendada sustentadas en evidencia.' },
]

export function LoginPage() {
  const { user, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('admin@energia.demo')
  const [password, setPassword] = useState('demo1234')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  if (user) return <Navigate to="/" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await login(email, password)
      const from = (location.state as { from?: string } | null)?.from ?? '/'
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'No se pudo iniciar sesión')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center px-4 py-10 text-white">
      <AuroraBackground />
      <div className="grid w-full max-w-5xl items-center gap-10 lg:grid-cols-[1.1fr_1fr]">
        <div className="hidden lg:block">
          <Logo />
          <h1 className="mt-10 text-4xl font-bold leading-tight tracking-tight text-shadow-glass">
            Convierte lecturas de medidores en <span className="bg-gradient-to-r from-[#06B6D4] to-[#8B5CF6] bg-clip-text text-transparent">decisiones operativas</span>.
          </h1>
          <p className="mt-4 max-w-lg text-ink-soft">Detecta, explica y prioriza anomalías de energía para saber qué investigar primero y por qué.</p>
          <ul className="mt-10 space-y-4">
            {highlights.map(({ icon: Icon, title, text }) => (
              <li key={title} className="flex gap-4">
                <span className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-white/20 bg-white/[0.08]">
                  <Icon className="size-5 text-aurora-cyan" aria-hidden />
                </span>
                <div>
                  <p className="font-semibold">{title}</p>
                  <p className="text-sm text-ink-soft/80">{text}</p>
                </div>
              </li>
            ))}
          </ul>
        </div>

        <GlassCard dense className="mx-auto w-full max-w-md p-8">
          <div className="lg:hidden">
            <Logo />
          </div>
          <h2 className="mt-6 text-2xl font-bold lg:mt-0">Iniciar sesión</h2>
          <p className="mt-1 text-sm text-ink-soft">Accede al centro de control energético.</p>
          <form className="mt-8 space-y-4" onSubmit={onSubmit} noValidate>
            <label className="block">
              <span className="text-sm font-medium text-ink-soft">Correo</span>
              <div className="relative mt-1.5">
                <Mail className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-muted" aria-hidden />
                <input
                  type="email"
                  autoComplete="username"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="h-11 w-full rounded-xl border border-white/20 bg-white/[0.06] pl-10 pr-3 text-sm text-white placeholder:text-ink-muted focus:outline-none focus:ring-2 focus:ring-[#06B6D4]"
                />
              </div>
            </label>
            <label className="block">
              <span className="text-sm font-medium text-ink-soft">Contraseña</span>
              <div className="relative mt-1.5">
                <Lock className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-muted" aria-hidden />
                <input
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="h-11 w-full rounded-xl border border-white/20 bg-white/[0.06] pl-10 pr-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-[#06B6D4]"
                />
              </div>
            </label>
            {error && (
              <p className="rounded-lg border border-status-critical/40 bg-status-critical/10 px-3 py-2 text-sm text-red-100" role="alert">
                {error}
              </p>
            )}
            <Button type="submit" variant="primary" className="h-11 w-full" loading={loading}>
              Entrar
            </Button>
          </form>
          <div className="mt-6 flex items-start gap-2 rounded-xl border border-white/10 bg-white/[0.04] p-3 text-xs text-ink-soft">
            <ShieldCheck className="size-4 shrink-0 text-aurora-cyan" aria-hidden />
            <span>
              Entorno demo: <strong className="text-white">admin@energia.demo</strong> / <strong className="text-white">demo1234</strong>
            </span>
          </div>
        </GlassCard>
      </div>
    </div>
  )
}
