import type { CharacterResponse } from '../../api/types'

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
  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border bg-bg-card p-5">
      <div className="flex items-center justify-between">
        <h3 className="font-medium text-text-primary">{character.name}</h3>
        <div className="flex gap-2">
          <button
            className="rounded border border-border px-3 py-1 text-xs text-text-secondary transition-colors hover:text-text-primary cursor-pointer"
            onClick={() => onEdit(character)}
          >
            Edit
          </button>
          <button
            className="rounded border border-border px-3 py-1 text-xs text-error transition-colors hover:bg-error/10 cursor-pointer"
            onClick={() => onDelete(character)}
          >
            Delete
          </button>
        </div>
      </div>

      {formatAttributes(character.attributes) && (
        <p className="text-xs text-text-tertiary">
          {formatAttributes(character.attributes)}
        </p>
      )}

      {character.notes && (
        <p className="text-xs text-text-tertiary">
          Notes: &ldquo;{character.notes}&rdquo;
        </p>
      )}
    </div>
  )
}
