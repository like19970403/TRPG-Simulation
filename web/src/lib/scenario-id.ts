/**
 * Generates the next available ID with the given prefix from an existing list.
 * Format: prefix_N where N is auto-incremented.
 * Examples: scene_1, scene_2, item_1, npc_1
 */
export function generateNextId(
  prefix: string,
  existingIds: string[],
): string {
  let maxNum = 0
  const pattern = new RegExp(`^${prefix}_(\\d+)$`)
  for (const id of existingIds) {
    const match = id.match(pattern)
    if (match) {
      const num = parseInt(match[1], 10)
      if (num > maxNum) maxNum = num
    }
  }
  return `${prefix}_${maxNum + 1}`
}
