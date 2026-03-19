import { Input } from '../../ui/input'
import { Textarea } from '../../ui/textarea'
import type { ProfileField } from '../../../data/rule-presets'

interface StepProfileProps {
  profileFields: ProfileField[]
  profileData: Record<string, string>
  onFieldChange: (key: string, value: string) => void
  freeNotes: string
  onFreeNotesChange: (value: string) => void
}

export function StepProfile({
  profileFields,
  profileData,
  onFieldChange,
  freeNotes,
  onFreeNotesChange,
}: StepProfileProps) {
  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-text-tertiary">
        填寫角色背景（全部選填），讓角色更有深度
      </p>

      {profileFields.map((field) => (
        <div key={field.key} className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-secondary">
            {field.label}
          </label>
          {field.type === 'textarea' ? (
            <Textarea
              rows={2}
              value={profileData[field.key] ?? ''}
              onChange={(e) => onFieldChange(field.key, e.target.value)}
              placeholder={field.placeholder}
            />
          ) : (
            <Input
              value={profileData[field.key] ?? ''}
              onChange={(e) => onFieldChange(field.key, e.target.value)}
              placeholder={field.placeholder}
            />
          )}
        </div>
      ))}

      <div className="flex flex-col gap-1 border-t border-border pt-3">
        <label className="text-xs font-medium text-text-secondary">
          其他筆記
        </label>
        <Textarea
          rows={2}
          value={freeNotes}
          onChange={(e) => onFreeNotesChange(e.target.value)}
          placeholder="自由記錄任何額外資訊..."
        />
      </div>
    </div>
  )
}
