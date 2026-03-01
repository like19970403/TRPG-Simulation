import { Input } from '../../ui/input'
import { Select } from '../../ui/select'
import type { Action } from '../../../api/types'

type ActionType = 'set_var' | 'reveal_item' | 'reveal_npc_field'

interface ActionEditorProps {
  action: Action
  onChange: (a: Action) => void
  onRemove: () => void
  allItemIds: string[]
  allNpcIds: string[]
  allNpcFieldKeys: Record<string, string[]>
  allVariableNames: string[]
}

function getActionType(action: Action): ActionType {
  if (action.set_var) return 'set_var'
  if (action.reveal_item) return 'reveal_item'
  return 'reveal_npc_field'
}

function makeDefaultAction(type: ActionType): Action {
  switch (type) {
    case 'set_var':
      return { set_var: { name: '', value: '' } }
    case 'reveal_item':
      return { reveal_item: { item_id: '', to: 'current_player' } }
    case 'reveal_npc_field':
      return {
        reveal_npc_field: {
          npc_id: '',
          field_key: '',
          to: 'current_player',
        },
      }
  }
}

export function ActionEditor({
  action,
  onChange,
  onRemove,
  allItemIds,
  allNpcIds,
  allNpcFieldKeys,
  allVariableNames,
}: ActionEditorProps) {
  const actionType = getActionType(action)

  const handleTypeChange = (newType: ActionType) => {
    if (newType !== actionType) {
      onChange(makeDefaultAction(newType))
    }
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-[#1A1A1A] p-3">
      <div className="flex items-end gap-2">
        <label className="flex flex-col gap-1">
          <span className="text-xs text-text-tertiary">類型</span>
          <Select
            value={actionType}
            onChange={(e) => handleTypeChange(e.target.value as ActionType)}
            className="w-44"
          >
            <option value="set_var">set_var</option>
            <option value="reveal_item">reveal_item</option>
            <option value="reveal_npc_field">reveal_npc_field</option>
          </Select>
        </label>

        <div className="flex flex-1 gap-2">
          {actionType === 'set_var' && action.set_var && (
            <>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">變數名稱</span>
                <Select
                  value={action.set_var.name}
                  onChange={(e) =>
                    onChange({
                      set_var: { ...action.set_var!, name: e.target.value },
                    })
                  }
                >
                  <option value="">-- 選擇變數 --</option>
                  {allVariableNames.map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">值</span>
                <Input
                  value={String(action.set_var.value ?? '')}
                  onChange={(e) =>
                    onChange({
                      set_var: { ...action.set_var!, value: e.target.value },
                    })
                  }
                  placeholder="值或 expr 表達式"
                />
              </label>
            </>
          )}

          {actionType === 'reveal_item' && action.reveal_item && (
            <>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">道具</span>
                <Select
                  value={action.reveal_item.item_id}
                  onChange={(e) =>
                    onChange({
                      reveal_item: {
                        ...action.reveal_item!,
                        item_id: e.target.value,
                      },
                    })
                  }
                >
                  <option value="">-- 選擇道具 --</option>
                  {allItemIds.map((id) => (
                    <option key={id} value={id}>
                      {id}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-text-tertiary">對象</span>
                <Select
                  value={action.reveal_item.to}
                  onChange={(e) =>
                    onChange({
                      reveal_item: {
                        ...action.reveal_item!,
                        to: e.target.value,
                      },
                    })
                  }
                  className="w-40"
                >
                  <option value="current_player">current_player</option>
                  <option value="all">all</option>
                </Select>
              </label>
            </>
          )}

          {actionType === 'reveal_npc_field' && action.reveal_npc_field && (
            <>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">NPC</span>
                <Select
                  value={action.reveal_npc_field.npc_id}
                  onChange={(e) =>
                    onChange({
                      reveal_npc_field: {
                        ...action.reveal_npc_field!,
                        npc_id: e.target.value,
                        field_key: '',
                      },
                    })
                  }
                >
                  <option value="">-- 選擇 NPC --</option>
                  {allNpcIds.map((id) => (
                    <option key={id} value={id}>
                      {id}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-1 flex-col gap-1">
                <span className="text-xs text-text-tertiary">欄位 Key</span>
                <Select
                  value={action.reveal_npc_field.field_key}
                  onChange={(e) =>
                    onChange({
                      reveal_npc_field: {
                        ...action.reveal_npc_field!,
                        field_key: e.target.value,
                      },
                    })
                  }
                >
                  <option value="">-- 選擇欄位 --</option>
                  {(
                    allNpcFieldKeys[action.reveal_npc_field.npc_id] ?? []
                  ).map((k) => (
                    <option key={k} value={k}>
                      {k}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-text-tertiary">對象</span>
                <Select
                  value={action.reveal_npc_field.to}
                  onChange={(e) =>
                    onChange({
                      reveal_npc_field: {
                        ...action.reveal_npc_field!,
                        to: e.target.value,
                      },
                    })
                  }
                  className="w-40"
                >
                  <option value="current_player">current_player</option>
                  <option value="all">all</option>
                </Select>
              </label>
            </>
          )}
        </div>

        <button
          type="button"
          onClick={onRemove}
          className="shrink-0 pb-2.5 text-sm text-text-tertiary transition-colors hover:text-error"
        >
          刪除
        </button>
      </div>
    </div>
  )
}
