import type { RulePreset, ProfileField } from '../../../data/rule-presets'

interface StepReviewProps {
  name: string
  preset: RulePreset
  attributes: Record<string, number>
  profileFields: ProfileField[]
  profileData: Record<string, string>
  freeNotes: string
  selectedSkills?: string[]
  selectedCultivation?: string
}

export function StepReview({
  name,
  preset,
  attributes,
  profileFields,
  profileData,
  freeNotes,
  selectedSkills,
  selectedCultivation,
}: StepReviewProps) {
  const filledFields = profileFields.filter(
    (f) => profileData[f.key]?.trim(),
  )

  const skillDefs = preset.martialSkills ?? []
  const cultivationDefs = preset.cultivationMethods ?? []

  const chosenSkills = skillDefs.filter((s) => selectedSkills?.includes(s.id))
  const chosenCultivation = cultivationDefs.find(
    (c) => c.id === selectedCultivation,
  )

  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-text-tertiary">確認角色資訊後點擊建立</p>

      <div className="rounded-xl border border-border bg-bg-card p-4">
        {/* Name + system badge */}
        <div className="mb-3 flex items-center gap-2">
          <h3 className="font-display text-lg font-bold text-text-primary">{name}</h3>
          <span className="rounded-full bg-gold/20 px-2 py-0.5 text-[10px] font-medium text-gold">
            {preset.name.split('（')[0]}
          </span>
        </div>

        {/* Attributes */}
        <div className="mb-3 flex flex-wrap gap-2">
          {(preset.rules.attributes ?? []).map((attr) => (
            <div
              key={attr.display}
              className="rounded-md border border-border px-2.5 py-1 text-xs"
            >
              <span className="text-text-tertiary">{attr.display}</span>{' '}
              <span className="font-medium text-text-primary">
                {attributes[attr.display] ?? attr.default}
              </span>
            </div>
          ))}
        </div>

        {/* Skills & Cultivation */}
        {(chosenSkills.length > 0 || chosenCultivation) && (
          <div className="mb-3 border-t border-border pt-3">
            {chosenSkills.length > 0 && (
              <div className="mb-2">
                <span className="text-[10px] font-medium text-text-tertiary">
                  起始武學
                </span>
                <div className="mt-1 flex flex-wrap gap-1.5">
                  {chosenSkills.map((skill) => (
                    <span
                      key={skill.id}
                      className="rounded-md border border-border px-2 py-0.5 text-[10px] text-text-secondary"
                    >
                      {skill.name}
                      <span className="ml-1 text-text-tertiary">
                        {skill.cost && `消耗${skill.cost}`}
                      </span>
                    </span>
                  ))}
                </div>
              </div>
            )}
            {chosenCultivation && (
              <div>
                <span className="text-[10px] font-medium text-text-tertiary">
                  起始心法
                </span>
                <div className="mt-1">
                  <span className="rounded-md border border-gold/30 bg-gold/10 px-2 py-0.5 text-[10px] text-gold">
                    {chosenCultivation.name}
                    {chosenCultivation.special && (
                      <span className="ml-1 text-amber-400">
                        ({chosenCultivation.special})
                      </span>
                    )}
                  </span>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Profile fields */}
        {filledFields.length > 0 && (
          <div className="flex flex-col gap-2 border-t border-border pt-3">
            {filledFields.map((field) => (
              <div key={field.key}>
                <span className="text-[10px] font-medium text-text-tertiary">
                  {field.label}
                </span>
                <p className="text-xs text-text-secondary whitespace-pre-wrap">
                  {profileData[field.key]}
                </p>
              </div>
            ))}
          </div>
        )}

        {/* Free notes */}
        {freeNotes.trim() && (
          <div className="mt-2 border-t border-border pt-2">
            <span className="text-[10px] font-medium text-text-tertiary">
              筆記
            </span>
            <p className="text-xs text-text-secondary whitespace-pre-wrap">
              {freeNotes}
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
