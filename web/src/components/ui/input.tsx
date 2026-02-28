import { type InputHTMLAttributes, forwardRef } from 'react'
import { cn } from '../../lib/cn'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  error?: boolean
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ error, className, ...props }, ref) => {
    return (
      <input
        ref={ref}
        className={cn(
          'w-full rounded-lg bg-bg-input px-3 py-2.5 text-sm text-text-primary',
          'border outline-none transition-colors',
          'placeholder:text-text-tertiary',
          error
            ? 'border-error focus:border-error'
            : 'border-border focus:border-border-focus',
          className,
        )}
        {...props}
      />
    )
  },
)

Input.displayName = 'Input'
