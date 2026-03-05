/** Action type labels */
export const ACTION_TYPE_LABELS: Record<string, string> = {
  set_var: '設定變數',
  reveal_item: '揭示道具',
  give_item: '給予道具',
  remove_item: '移除道具',
  reveal_npc_field: '揭示 NPC 資訊',
}

/** Trigger type labels */
export const TRIGGER_TYPE_LABELS: Record<string, string> = {
  player_choice: '玩家投票',
  gm_decision: 'GM 決定',
  auto: '自動觸發',
  condition_met: '條件達成',
}

/** Variable type labels */
export const VARIABLE_TYPE_LABELS: Record<string, string> = {
  bool: '布林值',
  int: '整數',
  string: '文字',
}

/** NPC field visibility labels (dropdown options — only canonical values) */
export const VISIBILITY_LABELS: Record<string, string> = {
  visible: '公開',
  hidden: '隱藏',
  gm_only: '僅 GM',
}

/** Reveal target labels */
export const REVEAL_TARGET_LABELS: Record<string, string> = {
  current_player: '當前玩家',
  all: '所有人',
}

/** Item type labels */
export const ITEM_TYPE_LABELS: Record<string, string> = {
  key_item: '關鍵道具',
  item: '道具',
  clue: '線索',
  evidence: '證物',
  consumable: '消耗品',
  treasure: '寶物',
  weapon: '武器',
  armor: '防具',
  tool: '工具',
}

/** Arithmetic operators for set_var expression builder */
export const ARITHMETIC_OPERATORS = [
  { value: '', label: '直接設值' },
  { value: '+', label: '加 (+)' },
  { value: '-', label: '減 (-)' },
  { value: '*', label: '乘 (*)' },
  { value: '/', label: '除 (/)' },
] as const

/** Condition operators for the visual builder */
export const CONDITION_OPERATORS = [
  { value: '==', label: '等於 (==)' },
  { value: '!=', label: '不等於 (!=)' },
  { value: '>', label: '大於 (>)' },
  { value: '>=', label: '大於等於 (>=)' },
  { value: '<', label: '小於 (<)' },
  { value: '<=', label: '小於等於 (<=)' },
] as const
