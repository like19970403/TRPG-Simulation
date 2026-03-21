import type { Item } from '../api/types'

/** Parse inner force cost from item's gm_notes. Supports both Chinese and English colons. */
export function parseSkillCost(item: Item | undefined): number {
  if (!item?.gm_notes) return 2
  const match = item.gm_notes.match(/消耗[:：]\s*(\d+)/)
  return match ? parseInt(match[1]) : 2
}

/** Parse skill bonus (武功 +N) from item's gm_notes. */
export function parseSkillBonus(item: Item | undefined): number {
  if (!item?.gm_notes) return 0
  const match = item.gm_notes.match(/武功\s*[+＋]\s*(\d+)/)
  return match ? parseInt(match[1]) : 0
}

/** Parse skill level from item's description (初級/中級/高級). */
export function parseSkillLevel(item: Item | undefined): string {
  if (!item?.description) return '初級'
  const match = item.description.match(/(初級|中級|高級)/)
  return match ? match[1] : '初級'
}
