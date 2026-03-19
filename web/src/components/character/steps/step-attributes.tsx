import type { RulePreset } from '../../../data/rule-presets'

interface StepAttributesProps {
  preset: RulePreset
  attributes: Record<string, number>
  onAttributeChange: (key: string, value: number) => void
}

const MIN_ATTR = 1
const MAX_ATTR = 10

export function StepAttributes({
  preset,
  attributes,
  onAttributeChange,
}: StepAttributesProps) {
  const totalPool = (preset.rules.attributes ?? []).reduce(
    (sum, a) => sum + a.default,
    0,
  )
  const currentTotal = Object.values(attributes).reduce((sum, v) => sum + v, 0)
  const remaining = totalPool - currentTotal

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-text-tertiary">
          分配屬性點數，總計 {totalPool} 點
        </p>
        <span
          className={`text-sm font-medium ${remaining === 0 ? 'text-green-400' : remaining < 0 ? 'text-error' : 'text-gold'}`}
        >
          剩餘：{remaining}
        </span>
      </div>

      <div className="flex flex-col gap-2.5">
        {(preset.rules.attributes ?? []).map((attr) => {
          const display = attr.display
          const value = attributes[display] ?? attr.default
          const desc = preset.attributeDescriptions[display]

          return (
            <div
              key={display}
              className="flex h-14 items-center justify-between rounded-[10px] border border-border bg-bg-card px-4"
            >
              <div className="flex-1">
                <div className="text-sm font-medium text-text-primary">
                  {display}
                </div>
                {desc && (
                  <div className="text-[10px] text-text-tertiary">{desc}</div>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() =>
                    onAttributeChange(display, Math.max(MIN_ATTR, value - 1))
                  }
                  disabled={value <= MIN_ATTR}
                  className="flex h-7 w-7 items-center justify-center rounded-md border border-border text-sm text-text-secondary transition-colors hover:border-gold hover:text-gold disabled:opacity-30"
                >
                  -
                </button>
                <span className="w-6 text-center text-sm font-medium text-text-primary">
                  {value}
                </span>
                <button
                  type="button"
                  onClick={() =>
                    onAttributeChange(display, Math.min(MAX_ATTR, value + 1))
                  }
                  disabled={value >= MAX_ATTR || remaining <= 0}
                  className="flex h-7 w-7 items-center justify-center rounded-md border border-border text-sm text-text-secondary transition-colors hover:border-gold hover:text-gold disabled:opacity-30"
                >
                  +
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
