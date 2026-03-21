import { cn } from '../../../lib/cn'
import type { SkillDefinition } from '../../../data/rule-presets'
import type { Item } from '../../../api/types'

interface StepSkillsProps {
  martialSkills: SkillDefinition[]
  cultivationMethods: SkillDefinition[]
  startingWeapons?: Item[]
  selectedWeapon: string
  selectedSkills: string[]
  selectedCultivation: string
  onWeaponChange: (id: string) => void
  onSkillsChange: (skills: string[]) => void
  onCultivationChange: (id: string) => void
  maxSkills: number
}

const LEVEL_COLORS: Record<string, string> = {
  '初級': 'bg-emerald-900/40 text-emerald-400',
  '中級': 'bg-blue-900/40 text-blue-400',
  '高級': 'bg-amber-900/40 text-amber-400',
}

const WEAPON_LABELS: Record<string, string> = {
  palm: '掌',
  blade: '刀',
  spear: '槍',
  sword: '劍',
  hidden: '暗器',
}

export function StepSkills({
  martialSkills,
  cultivationMethods,
  startingWeapons = [],
  selectedWeapon,
  selectedSkills,
  selectedCultivation,
  onWeaponChange,
  onSkillsChange,
  onCultivationChange,
  maxSkills,
}: StepSkillsProps) {
  // Get selected weapon's type for filtering skills
  const selectedWeaponType = startingWeapons.find((w) => w.id === selectedWeapon)?.weapon_type

  // Filter martial skills by selected weapon type
  const filteredSkills = selectedWeaponType
    ? martialSkills.filter((s) => s.weaponType === selectedWeaponType)
    : martialSkills

  const toggleSkill = (id: string) => {
    if (selectedSkills.includes(id)) {
      onSkillsChange(selectedSkills.filter((s) => s !== id))
    } else if (selectedSkills.length < maxSkills) {
      onSkillsChange([...selectedSkills, id])
    }
  }

  // When weapon changes, clear skills that don't match
  const handleWeaponChange = (weaponId: string) => {
    onWeaponChange(weaponId)
    const newType = startingWeapons.find((w) => w.id === weaponId)?.weapon_type
    const validSkills = selectedSkills.filter((sId) => {
      const skill = martialSkills.find((s) => s.id === sId)
      return skill?.weaponType === newType
    })
    if (validSkills.length !== selectedSkills.length) {
      onSkillsChange(validSkills)
    }
  }

  return (
    <div className="flex flex-col gap-5">
      {/* Weapon Selection */}
      {startingWeapons.length > 0 && (
        <div>
          <div className="mb-2 flex items-center justify-between">
            <h4 className="text-sm font-semibold text-text-primary">選擇武器</h4>
            <span className="text-xs text-text-tertiary">
              {selectedWeapon ? '已選 1/1' : '已選 0/1'}
            </span>
          </div>
          <p className="mb-3 text-[10px] text-text-tertiary">
            武器決定你可以使用的武學。選擇後武學列表會自動過濾
          </p>
          <div className="grid grid-cols-5 gap-2">
            {startingWeapons.map((weapon) => {
              const selected = selectedWeapon === weapon.id
              return (
                <button
                  key={weapon.id}
                  type="button"
                  onClick={() => handleWeaponChange(selected ? '' : weapon.id)}
                  className={cn(
                    'flex flex-col items-center gap-1 rounded-lg border p-3 text-center transition-all',
                    selected
                      ? 'border-gold bg-gold/10'
                      : 'border-border hover:border-text-tertiary',
                  )}
                >
                  {selected && <span className="text-xs text-gold">✓</span>}
                  <span className="text-xs font-semibold text-text-primary">{weapon.name}</span>
                  <span className="text-[9px] text-text-tertiary">
                    {WEAPON_LABELS[weapon.weapon_type ?? ''] ?? weapon.weapon_type}
                  </span>
                  <span className="text-[9px] text-text-tertiary">
                    atk +{weapon.atk ?? 0}
                  </span>
                </button>
              )
            })}
          </div>
        </div>
      )}

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
          {selectedWeaponType
            ? `顯示${WEAPON_LABELS[selectedWeaponType] ?? ''}類武學`
            : '請先選擇武器'}
        </p>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {filteredSkills.map((skill) => {
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
          {filteredSkills.length === 0 && selectedWeaponType && (
            <p className="text-xs text-text-tertiary col-span-2">此武器類型目前沒有可選武學</p>
          )}
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
