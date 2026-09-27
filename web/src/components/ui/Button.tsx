import clsx from 'clsx'
import type { ButtonHTMLAttributes, ReactNode } from 'react'

type Variant = 'primary' | 'secondary' | 'ghost'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: 'sm' | 'md'
  icon?: ReactNode
  loading?: boolean
}

const variants: Record<Variant, string> = {
  primary:
    'bg-gradient-to-r from-[#06B6D4] to-[#8B5CF6] text-white shadow-[0_8px_24px_-6px_rgba(139,92,246,0.55)] hover:brightness-110 disabled:opacity-60',
  secondary: 'border border-white/20 bg-white/[0.08] text-white hover:bg-white/[0.14] disabled:opacity-50',
  ghost: 'text-ink-soft hover:bg-white/[0.08] hover:text-white disabled:opacity-50',
}

export function Button({ variant = 'secondary', size = 'md', icon, loading, className, children, disabled, ...rest }: ButtonProps) {
  return (
    <button
      className={clsx(
        'inline-flex items-center justify-center gap-2 rounded-xl font-semibold transition focus:outline-none focus-visible:ring-2 focus-visible:ring-[#06B6D4] disabled:cursor-not-allowed',
        size === 'sm' ? 'h-8 px-3 text-xs' : 'h-10 px-4 text-sm',
        variants[variant],
        className,
      )}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {loading ? <span className="size-4 animate-spin rounded-full border-2 border-white/30 border-t-white" aria-hidden /> : icon}
      {children}
    </button>
  )
}
