import { useState } from 'react'
import { WizardStepIndicator } from './wizard-step-indicator'
import { StepBasicInfo } from './steps/step-basic-info'
import { StepAttributes } from './steps/step-attributes'
import { StepSkills } from './steps/step-skills'
import { StepProfile } from './steps/step-profile'
import { StepReview } from './steps/step-review'
import { RULE_PRESETS } from '../../data/rule-presets'
import { serializeProfile, parseProfile } from '../../lib/character-profile'
import { createCharacter, updateCharacter } from '../../api/characters'
import { ApiClientError } from '../../api/client'
import type { CharacterResponse } from '../../api/types'

interface CharacterWizardProps {
  character?: CharacterResponse | null
  onSaved: () => void
  onCancel: () => void
  onSwitchToLegacy: () => void
}

export function CharacterWizard({
  character,
  onSaved,
  onCancel,
  onSwitchToLegacy,
}: CharacterWizardProps) {
  const isEdit = !!character

  // Parse existing profile if editing
  const existingProfile = character?.notes
    ? parseProfile(character.notes)
    : null
  const existingSystem = existingProfile?._system ?? ''

  // Resolve preset from existing profile or default
  const resolvePreset = (id: string) =>
    RULE_PRESETS.find((p) => p.id === id) ?? null

  const [step, setStep] = useState(0)
  const [name, setName] = useState(character?.name ?? '')
  const [avatarUrl, setAvatarUrl] = useState<string | undefined>(
    (existingProfile?._avatarUrl as string) ?? undefined
  )
  const [systemId, setSystemId] = useState(existingSystem)
  const [attributes, setAttributes] = useState<Record<string, number>>(() => {
    if (character?.attributes && Object.keys(character.attributes).length > 0) {
      const result: Record<string, number> = {}
      for (const [k, v] of Object.entries(character.attributes)) {
        result[k] = typeof v === 'number' ? v : Number(v) || 0
      }
      return result
    }
    return {}
  })
  const [profileData, setProfileData] = useState<Record<string, string>>(() => {
    if (!existingProfile) return {}
    const result: Record<string, string> = {}
    for (const [k, v] of Object.entries(existingProfile)) {
      if (k.startsWith('_')) continue
      if (typeof v === 'string') result[k] = v
    }
    return result
  })
  const [freeNotes, setFreeNotes] = useState(
    (existingProfile?._freeNotes as string) ?? '',
  )
  const [selectedSkills, setSelectedSkills] = useState<string[]>(
    (existingProfile?._startingSkills as string[]) ?? [],
  )
  const [selectedCultivation, setSelectedCultivation] = useState(
    (existingProfile?._startingCultivation as string) ?? '',
  )
  const [selectedWeapon, setSelectedWeapon] = useState(
    (existingProfile?._startingWeapon as string) ?? '',
  )
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const preset = resolvePreset(systemId)
  const hasSkillsStep = !!(preset?.martialSkills?.length)

  // Dynamic step labels
  const stepLabels = hasSkillsStep
    ? ['基本資訊', '屬性配置', '武學心法', '角色檔案', '確認建立']
    : ['基本資訊', '屬性配置', '角色檔案', '確認建立']
  const totalSteps = stepLabels.length

  // Map logical step index to content
  // For wuxia: 0=basic, 1=attrs, 2=skills, 3=profile, 4=review
  // For detective: 0=basic, 1=attrs, 2=profile, 3=review
  const getStepContent = () => {
    if (hasSkillsStep) {
      return step // direct mapping for 5-step flow
    }
    // 4-step flow: skip skills (step 2 in 5-step)
    if (step >= 2) return step + 1 // shift to skip skills
    return step
  }
  const contentStep = getStepContent()

  // Initialize attributes when system changes
  const handleSystemChange = (newSystemId: string) => {
    setSystemId(newSystemId)
    if (newSystemId === 'custom') return
    const newPreset = resolvePreset(newSystemId)
    if (newPreset?.rules.attributes) {
      const newAttrs: Record<string, number> = {}
      for (const attr of newPreset.rules.attributes) {
        newAttrs[attr.display] = attr.default
      }
      setAttributes(newAttrs)
    }
    setProfileData({})
    setFreeNotes('')
    setSelectedWeapon('')
    setSelectedSkills([])
    setSelectedCultivation('')
  }

  const handleAttributeChange = (key: string, value: number) => {
    setAttributes((prev) => ({ ...prev, [key]: value }))
  }

  const handleFieldChange = (key: string, value: string) => {
    setProfileData((prev) => ({ ...prev, [key]: value }))
  }

  const canProceed = (): boolean => {
    if (step === 0) return name.trim().length > 0 && systemId !== ''
    if (step === 1 && preset) {
      const total = (preset.rules.attributes ?? []).reduce(
        (s, a) => s + a.default,
        0,
      )
      const current = Object.values(attributes).reduce((s, v) => s + v, 0)
      return current === total
    }
    return true
  }

  const handleSubmit = async () => {
    if (!preset) return
    setLoading(true)
    setError('')

    const notes = serializeProfile(
      systemId,
      profileData,
      freeNotes,
      avatarUrl,
      selectedSkills.length > 0 ? selectedSkills : undefined,
      selectedCultivation || undefined,
      selectedWeapon || undefined,
    )

    // Build initial inventory from selected weapon/skills/cultivation + inner force points
    const buildInitialInventory = () => {
      const inv: Array<{ item_id: string; quantity: number }> = []
      if (selectedWeapon) {
        inv.push({ item_id: selectedWeapon, quantity: 1 })
      }
      for (const skillId of selectedSkills) {
        inv.push({ item_id: skillId, quantity: 1 })
      }
      if (selectedCultivation) {
        inv.push({ item_id: selectedCultivation, quantity: 1 })
      }
      // Inner force points = 內力 attribute value
      const innerForceAttr = attributes['內力'] ?? 5
      if (innerForceAttr > 0) {
        inv.push({ item_id: 'inner_force_point', quantity: innerForceAttr })
      }
      return inv
    }

    try {
      if (isEdit && character) {
        await updateCharacter(character.id, {
          name: name.trim(),
          attributes,
          inventory: character.inventory,
          notes,
        })
      } else {
        await createCharacter({
          name: name.trim(),
          attributes,
          inventory: buildInitialInventory(),
          notes,
        })
      }
      onSaved()
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message)
      } else {
        setError('儲存失敗')
      }
    } finally {
      setLoading(false)
    }
  }

  // Custom system → switch to legacy form
  if (systemId === 'custom') {
    onSwitchToLegacy()
    return null
  }

  const isLastStep = step === totalSteps - 1

  return (
    <div className="flex flex-col gap-2">
      <WizardStepIndicator currentStep={step} steps={stepLabels} />

      <div className="min-h-70 px-1">
        {contentStep === 0 && (
          <StepBasicInfo
            name={name}
            onNameChange={setName}
            systemId={systemId}
            onSystemChange={handleSystemChange}
            avatarUrl={avatarUrl}
            onAvatarChange={setAvatarUrl}
          />
        )}

        {contentStep === 1 && preset && (
          <StepAttributes
            preset={preset}
            attributes={attributes}
            onAttributeChange={handleAttributeChange}
          />
        )}

        {contentStep === 2 && preset?.martialSkills && preset?.cultivationMethods && (
          <StepSkills
            martialSkills={preset.martialSkills}
            cultivationMethods={preset.cultivationMethods}
            startingWeapons={preset.startingWeapons}
            selectedWeapon={selectedWeapon}
            selectedSkills={selectedSkills}
            selectedCultivation={selectedCultivation}
            onWeaponChange={setSelectedWeapon}
            onSkillsChange={setSelectedSkills}
            onCultivationChange={setSelectedCultivation}
            maxSkills={preset.startingSkillSlots ?? 2}
          />
        )}

        {contentStep === 3 && preset && (
          <StepProfile
            profileFields={preset.profileFields}
            profileData={profileData}
            onFieldChange={handleFieldChange}
            freeNotes={freeNotes}
            onFreeNotesChange={setFreeNotes}
          />
        )}

        {contentStep === 4 && preset && (
          <StepReview
            name={name}
            preset={preset}
            attributes={attributes}
            profileFields={preset.profileFields}
            profileData={profileData}
            freeNotes={freeNotes}
            selectedWeapon={selectedWeapon}
            selectedSkills={selectedSkills}
            selectedCultivation={selectedCultivation}
          />
        )}
      </div>

      {error && <p className="text-xs text-error">{error}</p>}

      {/* Navigation buttons */}
      <div className="flex items-center justify-between border-t border-border pt-3">
        <div>
          {step === 0 && !isEdit && (
            <button
              type="button"
              onClick={onSwitchToLegacy}
              className="text-[10px] text-text-tertiary underline transition-colors hover:text-text-secondary"
            >
              進階模式
            </button>
          )}
        </div>
        <div className="flex gap-2">
          {step === 0 ? (
            <button
              type="button"
              onClick={onCancel}
              className="flex h-9 items-center justify-center rounded-lg border border-border px-5 text-[13px] text-text-secondary transition-colors hover:text-text-primary"
            >
              取消
            </button>
          ) : (
            <button
              type="button"
              onClick={() => setStep(step - 1)}
              className="flex h-9 items-center justify-center rounded-lg border border-border px-5 text-[13px] text-text-secondary transition-colors hover:text-text-primary"
            >
              上一步
            </button>
          )}

          {!isLastStep ? (
            <button
              type="button"
              onClick={() => setStep(step + 1)}
              disabled={!canProceed()}
              className="flex h-9 items-center justify-center rounded-lg bg-gold px-5 text-[13px] font-semibold text-bg-page transition-colors disabled:opacity-40"
            >
              下一步
            </button>
          ) : (
            <button
              type="button"
              onClick={handleSubmit}
              disabled={loading}
              className="flex h-9 items-center justify-center rounded-lg bg-gold px-5 text-[13px] font-semibold text-bg-page transition-colors disabled:opacity-40"
            >
              {loading ? '建立中...' : isEdit ? '儲存變更' : '建立角色'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
