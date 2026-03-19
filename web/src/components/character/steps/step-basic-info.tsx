import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import { ImageUpload } from '../../ui/image-upload'
import { RULE_PRESETS } from '../../../data/rule-presets'

interface StepBasicInfoProps {
  name: string
  onNameChange: (name: string) => void
  systemId: string
  onSystemChange: (systemId: string) => void
  avatarUrl?: string
  onAvatarChange: (url: string | undefined) => void
}

export function StepBasicInfo({
  name,
  onNameChange,
  systemId,
  onSystemChange,
  avatarUrl,
  onAvatarChange,
}: StepBasicInfoProps) {
  const preset = RULE_PRESETS.find((p) => p.id === systemId)

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-text-secondary">
          角色名稱 <span className="text-error">*</span>
        </label>
        <Input
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder="為你的角色取個名字"
          maxLength={100}
          autoFocus
        />
      </div>

      <div className="flex flex-col gap-1">
        <ImageUpload
          value={avatarUrl}
          onChange={onAvatarChange}
          label="角色頭像"
          previewClass="h-24 w-24 rounded-xl object-cover"
        />
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium text-text-secondary">
          規則系統 <span className="text-error">*</span>
        </label>
        <Select
          value={systemId}
          onChange={(e) => onSystemChange(e.target.value)}
        >
          <option value="">-- 請選擇 --</option>
          {RULE_PRESETS.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
          <option value="custom">自訂（不使用模板）</option>
        </Select>
        {preset && (
          <p className="text-xs text-text-tertiary">{preset.description}</p>
        )}
        {systemId === 'custom' && (
          <p className="text-xs text-text-tertiary">
            選擇「自訂」將使用簡易表單，可自由定義屬性
          </p>
        )}
      </div>
    </div>
  )
}
