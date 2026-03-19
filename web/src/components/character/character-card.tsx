import type { CharacterResponse } from '../../api/types'
import { getProfileSummary } from '../../lib/character-profile'
import { RULE_PRESETS } from '../../data/rule-presets'

interface CharacterCardProps {
  character: CharacterResponse
  onEdit: (character: CharacterResponse) => void
  onDelete: (character: CharacterResponse) => void
}

function formatAttributes(attrs: Record<string, unknown>): string {
  const entries = Object.entries(attrs).slice(0, 4)
  if (entries.length === 0) return ''
  return entries.map(([k, v]) => `${k}: ${v}`).join(' · ')
}

export function CharacterCard({
  character,
  onEdit,
  onDelete,
}: CharacterCardProps) {
  const profileSummary = getProfileSummary(character.notes)
  const preset = profileSummary?.system
    ? RULE_PRESETS.find((p) => p.id === profileSummary.system)
    : null

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border bg-bg-card p-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="font-medium text-text-primary">{character.name}</h3>
          {preset && (
            <span className="rounded-full bg-gold/20 px-2 py-0.5 text-[10px] font-medium text-gold">
              {preset.name.split('（')[0]}
            </span>
          )}
        </div>
        <div className="flex gap-2">
          <button
            className="rounded border border-border px-3 py-1 text-xs text-text-secondary transition-colors hover:text-text-primary cursor-pointer"
            onClick={() => onEdit(character)}
          >
            編輯
          </button>
          <button
            className="rounded border border-border px-3 py-1 text-xs text-error transition-colors hover:bg-error/10 cursor-pointer"
            onClick={() => onDelete(character)}
          >
            刪除
          </button>
        </div>
      </div>

      {profileSummary?.subtitle && (
        <p className="text-xs text-text-secondary">
          {profileSummary.subtitle}
        </p>
      )}

      {formatAttributes(character.attributes) && (
        <p className="text-xs text-text-tertiary">
          {formatAttributes(character.attributes)}
        </p>
      )}

      {!profileSummary && character.notes && (
        <p className="text-xs text-text-tertiary">
          筆記：&ldquo;{character.notes}&rdquo;
        </p>
      )}
    </div>
  )
}
