import clsx from 'clsx'
import type { HTMLAttributes, ReactNode } from 'react'

interface GlassCardProps extends HTMLAttributes<HTMLDivElement> {
  dense?: boolean
  children: ReactNode
}

export function GlassCard({ dense = false, className, children, ...rest }: GlassCardProps) {
  return (
    <div
      className={clsx(
        'rounded-2xl border border-white/20 shadow-[0_8px_32px_0_rgba(0,0,0,0.37)]',
        dense ? 'bg-[#0F172A]/85 backdrop-blur-[24px]' : 'bg-white/[0.08] backdrop-blur-[16px]',
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  )
}

interface SectionProps {
  title: string
  description?: string
  action?: ReactNode
  dense?: boolean
  className?: string
  bodyClassName?: string
  children: ReactNode
}

export function Section({ title, description, action, dense, className, bodyClassName, children }: SectionProps) {
  return (
    <GlassCard dense={dense} className={clsx('flex flex-col', className)}>
      <div className="flex flex-wrap items-start justify-between gap-3 px-5 pt-5">
        <div>
          <h2 className="text-base font-semibold text-white text-shadow-glass">{title}</h2>
          {description && <p className="mt-0.5 text-sm text-ink-soft/80">{description}</p>}
        </div>
        {action}
      </div>
      <div className={clsx('flex-1 p-5', bodyClassName)}>{children}</div>
    </GlassCard>
  )
}
