import { type TextareaHTMLAttributes, forwardRef } from 'react'
import { cn } from '../../lib/cn'

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  error?: boolean
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ error, className, ...props }, ref) => {
    return (
      <textarea
        ref={ref}
        className={cn(
          'w-full rounded-lg bg-bg-input px-3 py-2.5 text-sm text-text-primary',
          'border outline-none transition-colors resize-y',
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

Textarea.displayName = 'Textarea'
