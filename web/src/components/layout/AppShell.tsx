import clsx from 'clsx'
import { Activity, BrainCircuit, Gauge, LayoutDashboard, LogOut, Menu, X } from 'lucide-react'
import { useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { useAnalysis } from '../../lib/analysis'
import { useAuth } from '../../lib/auth'
import { formatRelative } from '../../lib/format'
import { AnalysisPanel } from '../analysis/AnalysisPanel'
import { RunAnalysisButton } from '../analysis/RunAnalysisButton'
import { AuroraBackground } from './AuroraBackground'
import { Logo } from './Logo'

const nav = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/meters', label: 'Medidores', icon: Gauge },
  { to: '/anomalies', label: 'Anomalías IA', icon: BrainCircuit },
]

function NavItems({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <nav className="flex flex-col gap-1" aria-label="Principal">
      {nav.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          onClick={onNavigate}
          className={({ isActive }) =>
            clsx(
              'flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold transition focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]',
              isActive ? 'bg-white/[0.12] text-white shadow-[inset_0_0_0_1px_rgba(255,255,255,0.12)]' : 'text-ink-soft hover:bg-white/[0.06] hover:text-white',
            )
          }
        >
          <Icon className="size-4.5" aria-hidden />
          {label}
        </NavLink>
      ))}
    </nav>
  )
}

function LastAnalysis() {
  const { run, running, openPanel } = useAnalysis()
  return (
    <button
      onClick={openPanel}
      className="w-full rounded-2xl border border-white/15 bg-white/[0.05] p-3 text-left transition hover:bg-white/[0.09] focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]"
    >
      <div className="flex items-center gap-2 text-xs font-semibold text-ink-soft">
        <Activity className={clsx('size-3.5', running ? 'animate-pulse text-aurora-cyan' : 'text-ink-muted')} aria-hidden />
        Último análisis IA
      </div>
      <p className="mt-1 text-sm font-semibold text-white">{running ? 'En ejecución…' : run ? formatRelative(run.finished_at ?? run.started_at) : 'Aún no ejecutado'}</p>
      {run && !running && <p className="mt-0.5 line-clamp-2 text-xs text-ink-soft/80">{run.summary}</p>}
    </button>
  )
}

export function AppShell() {
  const { user, logout } = useAuth()
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <div className="min-h-screen text-white">
      <AuroraBackground />
      <a href="#main" className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-white focus:px-3 focus:py-2 focus:text-[#0F172A]">
        Saltar al contenido
      </a>

      <aside className="fixed inset-y-0 left-0 z-30 hidden w-64 flex-col gap-6 border-r border-white/10 bg-white/[0.04] p-5 backdrop-blur-[16px] lg:flex">
        <Logo />
        <NavItems />
        <div className="mt-auto space-y-3">
          <LastAnalysis />
          <div className="flex items-center gap-3 rounded-2xl border border-white/10 p-3">
            <div className="flex size-9 items-center justify-center rounded-full bg-gradient-to-br from-[#06B6D4] to-[#8B5CF6] text-sm font-bold">
              {user?.name?.[0] ?? 'U'}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold">{user?.name}</p>
              <p className="truncate text-xs text-ink-muted">{user?.role}</p>
            </div>
            <button onClick={logout} className="rounded-lg p-1.5 text-ink-soft hover:bg-white/10 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]" aria-label="Cerrar sesión">
              <LogOut className="size-4" aria-hidden />
            </button>
          </div>
        </div>
      </aside>

      <div className="lg:pl-64">
        <header className="sticky top-0 z-20 border-b border-white/10 bg-[#0F172A]/70 backdrop-blur-[16px]">
          <div className="mx-auto flex h-16 max-w-7xl items-center justify-between gap-3 px-4 sm:px-6 lg:px-8">
            <div className="flex items-center gap-3 lg:hidden">
              <button onClick={() => setMobileOpen(true)} className="rounded-lg p-2 hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4]" aria-label="Abrir menú">
                <Menu className="size-5" aria-hidden />
              </button>
              <Logo compact />
            </div>
            <div className="hidden text-sm text-ink-soft lg:block">
              Planta industrial · <span className="text-white">12 medidores</span> · Sep 2026
            </div>
            <RunAnalysisButton size="sm" />
          </div>
        </header>
        <main id="main" className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
          <Outlet />
        </main>
      </div>

      {mobileOpen && (
        <div className="fixed inset-0 z-40 lg:hidden" role="dialog" aria-modal="true">
          <button className="absolute inset-0 bg-black/60" onClick={() => setMobileOpen(false)} aria-label="Cerrar menú" />
          <div className="relative flex h-full w-72 flex-col gap-6 border-r border-white/15 bg-[#0F172A]/95 p-5 backdrop-blur-[24px]">
            <div className="flex items-center justify-between">
              <Logo />
              <button onClick={() => setMobileOpen(false)} className="rounded-lg p-2 hover:bg-white/10" aria-label="Cerrar menú">
                <X className="size-5" aria-hidden />
              </button>
            </div>
            <NavItems onNavigate={() => setMobileOpen(false)} />
            <div className="mt-auto space-y-3">
              <LastAnalysis />
              <button onClick={logout} className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-sm text-ink-soft hover:bg-white/10">
                <LogOut className="size-4" aria-hidden /> Cerrar sesión
              </button>
            </div>
          </div>
        </div>
      )}

      <AnalysisPanel />
    </div>
  )
}
