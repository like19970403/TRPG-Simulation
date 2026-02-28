import { type ButtonHTMLAttributes } from 'react'
import { cn } from '../../lib/cn'
import { LoadingSpinner } from './loading-spinner'

type Variant = 'primary' | 'secondary' | 'ghost'
type Size = 'sm' | 'md' | 'lg'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
}

const variantStyles: Record<Variant, string> = {
  primary:
    'bg-gold text-text-on-gold hover:bg-gold-light active:bg-gold-dark disabled:opacity-50',
  secondary:
    'border border-gold text-gold hover:bg-gold-tint-30 active:bg-gold-tint disabled:opacity-50',
  ghost:
    'text-text-secondary hover:text-text-primary hover:bg-gold-tint disabled:opacity-50',
}

const sizeStyles: Record<Size, string> = {
  sm: 'px-3 py-1.5 text-sm',
  md: 'px-4 py-2.5 text-sm',
  lg: 'px-6 py-3 text-base',
}

export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  className,
  disabled,
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        'inline-flex items-center justify-center rounded-lg font-medium transition-colors cursor-pointer',
        variantStyles[variant],
        sizeStyles[size],
        className,
      )}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? <LoadingSpinner className="mr-2 h-4 w-4" /> : null}
      {children}
    </button>
  )
}
