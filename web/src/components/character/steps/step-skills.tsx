import { cn } from '../../../lib/cn'
import type { SkillDefinition } from '../../../data/rule-presets'

interface StepSkillsProps {
  martialSkills: SkillDefinition[]
  cultivationMethods: SkillDefinition[]
  selectedSkills: string[]
  selectedCultivation: string
  onSkillsChange: (skills: string[]) => void
  onCultivationChange: (id: string) => void
  maxSkills: number
}

const LEVEL_COLORS: Record<string, string> = {
  '初級': 'bg-emerald-900/40 text-emerald-400',
  '中級': 'bg-blue-900/40 text-blue-400',
  '高級': 'bg-amber-900/40 text-amber-400',
}

export function StepSkills({
  martialSkills,
  cultivationMethods,
  selectedSkills,
  selectedCultivation,
  onSkillsChange,
  onCultivationChange,
  maxSkills,
}: StepSkillsProps) {
  const toggleSkill = (id: string) => {
    if (selectedSkills.includes(id)) {
      onSkillsChange(selectedSkills.filter((s) => s !== id))
    } else if (selectedSkills.length < maxSkills) {
      onSkillsChange([...selectedSkills, id])
    }
  }

  return (
    <div className="flex flex-col gap-5">
      {/* Martial Skills */}
      <div>
        <div className="mb-2 flex items-center justify-between">
          <h4 className="text-sm font-semibold text-text-primary">
            起始武學
          </h4>
          <span className="text-xs text-text-tertiary">
            已選 {selectedSkills.length}/{maxSkills}
          </span>
        </div>
        <p className="mb-3 text-[10px] text-text-tertiary">
          武學為主動技能，戰鬥中使用需消耗內力點，每場戰鬥結束後內力回滿
        </p>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {martialSkills.map((skill) => {
            const selected = selectedSkills.includes(skill.id)
            const disabled = !selected && selectedSkills.length >= maxSkills
            return (
              <button
                key={skill.id}
                type="button"
                onClick={() => toggleSkill(skill.id)}
                disabled={disabled}
                className={cn(
                  'flex flex-col gap-1 rounded-lg border p-3 text-left transition-all',
                  selected
                    ? 'border-gold bg-gold/10'
                    : disabled
                      ? 'cursor-not-allowed border-border opacity-40'
                      : 'border-border hover:border-text-tertiary',
                )}
              >
                <div className="flex items-center gap-2">
                  {selected && (
                    <span className="text-xs text-gold">✓</span>
                  )}
                  <span className="text-xs font-semibold text-text-primary">
                    {skill.name}
                  </span>
                  <span className={cn('rounded px-1.5 py-0.5 text-[9px] font-medium', LEVEL_COLORS[skill.level])}>
                    {skill.level}
                  </span>
                  {skill.cost && (
                    <span className="ml-auto text-[10px] text-text-tertiary">
                      消耗 {skill.cost} 內力
                    </span>
                  )}
                </div>
                <span className="text-[10px] text-text-secondary">
                  {skill.effect}
                </span>
              </button>
            )
          })}
        </div>
      </div>

      {/* Cultivation Methods */}
      <div>
        <div className="mb-2 flex items-center justify-between">
          <h4 className="text-sm font-semibold text-text-primary">
            起始心法
          </h4>
          <span className="text-xs text-text-tertiary">
            {selectedCultivation ? '已選 1/1' : '已選 0/1'}
          </span>
        </div>
        <p className="mb-3 text-[10px] text-text-tertiary">
          心法為被動技能，一次只能運行一個，提供屬性加成。高級心法有特殊能力
        </p>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {cultivationMethods.map((method) => {
            const selected = selectedCultivation === method.id
            return (
              <button
                key={method.id}
                type="button"
                onClick={() => onCultivationChange(selected ? '' : method.id)}
                className={cn(
                  'flex flex-col gap-1 rounded-lg border p-3 text-left transition-all',
                  selected
                    ? 'border-gold bg-gold/10'
                    : 'border-border hover:border-text-tertiary',
                )}
              >
                <div className="flex items-center gap-2">
                  {selected && (
                    <span className="text-xs text-gold">✓</span>
                  )}
                  <span className="text-xs font-semibold text-text-primary">
                    {method.name}
                  </span>
                  <span className={cn('rounded px-1.5 py-0.5 text-[9px] font-medium', LEVEL_COLORS[method.level])}>
                    {method.level}
                  </span>
                </div>
                <span className="text-[10px] text-text-secondary">
                  {method.effect}
                </span>
                {method.special && (
                  <span className="text-[10px] text-amber-400">
                    特殊：{method.special}
                  </span>
                )}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
