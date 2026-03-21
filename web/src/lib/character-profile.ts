export interface CharacterProfile {
  _version: number
  _system: string
  _freeNotes?: string
  _avatarUrl?: string
  _startingSkills?: string[]
  _startingCultivation?: string
  [key: string]: unknown
}

export function isStructuredProfile(notes: string): boolean {
  if (!notes.startsWith('{')) return false
  try {
    const parsed = JSON.parse(notes)
    return parsed._version === 1 && typeof parsed._system === 'string'
  } catch {
    return false
  }
}

export function parseProfile(notes: string): CharacterProfile | null {
  if (!isStructuredProfile(notes)) return null
  try {
    return JSON.parse(notes) as CharacterProfile
  } catch {
    return null
  }
}

export function serializeProfile(
  system: string,
  fields: Record<string, string>,
  freeNotes?: string,
  avatarUrl?: string,
  startingSkills?: string[],
  startingCultivation?: string,
  startingWeapon?: string,
): string {
  const profile: CharacterProfile = {
    _version: 1,
    _system: system,
    ...fields,
  }
  if (freeNotes?.trim()) {
    profile._freeNotes = freeNotes.trim()
  }
  if (avatarUrl?.trim()) {
    profile._avatarUrl = avatarUrl.trim()
  }
  if (startingSkills && startingSkills.length > 0) {
    profile._startingSkills = startingSkills
  }
  if (startingCultivation) {
    profile._startingCultivation = startingCultivation
  }
  if (startingWeapon) {
    profile._startingWeapon = startingWeapon
  }
  return JSON.stringify(profile)
}

export function getProfileSummary(
  notes: string,
): { system?: string; subtitle?: string } | null {
  const profile = parseProfile(notes)
  if (!profile) return null
  const system = profile._system as string
  // Pick the first non-empty text field as subtitle
  const skipKeys = new Set(['_version', '_system', '_freeNotes', '_avatarUrl', '_startingSkills', '_startingCultivation', '_startingWeapon'])
  for (const [key, value] of Object.entries(profile)) {
    if (skipKeys.has(key)) continue
    if (typeof value === 'string' && value.trim()) {
      return { system, subtitle: value.trim() }
    }
  }
  return { system }
}
