import { type SelectHTMLAttributes, forwardRef } from 'react'
import { cn } from '../../lib/cn'

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  error?: boolean
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ error, className, children, ...props }, ref) => {
    return (
      <select
        ref={ref}
        className={cn(
          'w-full rounded-lg bg-bg-input px-3 py-2.5 text-sm text-text-primary',
          'border outline-none transition-colors',
          'appearance-none cursor-pointer',
          error
            ? 'border-error focus:border-error'
            : 'border-border focus:border-border-focus',
          className,
        )}
        {...props}
      >
        {children}
      </select>
    )
  },
)

Select.displayName = 'Select'
