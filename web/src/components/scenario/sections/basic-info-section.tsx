import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import type { ScenarioContent } from '../../../api/types'

interface BasicInfoSectionProps {
  data: ScenarioContent
  onChange: (d: Partial<ScenarioContent>) => void
}

export function BasicInfoSection({ data, onChange }: BasicInfoSectionProps) {
  const sceneIds = data.scenes.map((s) => s.id).filter(Boolean)

  return (
    <div className="flex flex-col gap-4">
      <label className="flex flex-col gap-1">
        <span className="text-xs font-medium text-text-secondary">
          內容 ID
        </span>
        <Input
          value={data.id}
          onChange={(e) => onChange({ id: e.target.value })}
          placeholder="劇本內容 ID"
        />
      </label>

      <label className="flex flex-col gap-1">
        <span className="text-xs font-medium text-text-secondary">
          內容標題
        </span>
        <Input
          value={data.title}
          onChange={(e) => onChange({ title: e.target.value })}
          placeholder="劇本內容標題"
        />
      </label>

      <label className="flex flex-col gap-1">
        <span className="text-xs font-medium text-text-secondary">
          起始場景
        </span>
        <Select
          value={data.start_scene}
          onChange={(e) => onChange({ start_scene: e.target.value })}
        >
          <option value="">
            {sceneIds.length === 0 ? '請先新增場景' : '-- 選擇起始場景 --'}
          </option>
          {sceneIds.map((id) => (
            <option key={id} value={id}>
              {id}
            </option>
          ))}
        </Select>
      </label>
    </div>
  )
}
