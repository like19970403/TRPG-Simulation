import type { ReactNode } from 'react'

interface FormFieldProps {
  label: string
  error?: string
  children: ReactNode
}

export function FormField({ label, error, children }: FormFieldProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-xs text-text-secondary">{label}</label>
      {children}
      {error ? <p className="text-xs text-error">{error}</p> : null}
    </div>
  )
}
