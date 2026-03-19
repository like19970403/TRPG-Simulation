import { cn } from '../../lib/cn'

interface WizardStepIndicatorProps {
  currentStep: number
  steps: string[]
}

export function WizardStepIndicator({ currentStep, steps }: WizardStepIndicatorProps) {
  return (
    <div className="flex items-center justify-center px-4 py-3">
      {steps.map((label, i) => (
        <div key={label} className="flex items-center">
          <div className="flex flex-col items-center gap-1">
            <div
              className={cn(
                'flex h-7 w-7 items-center justify-center rounded-full text-xs font-semibold transition-colors',
                i < currentStep
                  ? 'border-2 border-gold text-gold'
                  : i === currentStep
                    ? 'bg-gold text-bg-page'
                    : 'border-2 border-border text-text-tertiary',
              )}
            >
              {i < currentStep ? '✓' : i + 1}
            </div>
            <span
              className={cn(
                'text-[9px] whitespace-nowrap',
                i <= currentStep ? 'text-gold' : 'text-text-tertiary',
              )}
            >
              {label}
            </span>
          </div>
          {i < steps.length - 1 && (
            <div
              className={cn(
                'mx-1 mb-5 h-px w-8',
                i < currentStep ? 'bg-gold' : 'bg-border',
              )}
            />
          )}
        </div>
      ))}
    </div>
  )
}
