export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface RegisterResponse {
  id: string
  username: string
  email: string
  createdAt: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface TokenResponse {
  accessToken: string
  expiresIn: number
  tokenType: string
}

export interface ErrorDetail {
  field: string
  reason: string
}

export interface ApiError {
  error: string
  message: string
  details?: ErrorDetail[]
}

export interface User {
  id: string
  username: string
}

// --- Scenario types ---

export type ScenarioStatus = 'draft' | 'published' | 'archived'

export interface ScenarioResponse {
  id: string
  authorId: string
  title: string
  description: string
  version: number
  status: ScenarioStatus
  content: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface ScenarioListResponse {
  scenarios: ScenarioResponse[]
  total: number
  limit: number
  offset: number
}

export interface CreateScenarioRequest {
  title: string
  description: string
  content: Record<string, unknown>
}

export interface UpdateScenarioRequest {
  title: string
  description: string
  content: Record<string, unknown>
}

// --- Session types (REST API — camelCase JSON, matching Go json tags) ---

export type SessionStatus = 'lobby' | 'active' | 'paused' | 'completed'

export interface SessionResponse {
  id: string
  scenarioId: string
  gmId: string
  status: SessionStatus
  inviteCode: string
  createdAt: string
  startedAt: string | null
  endedAt: string | null
}

export interface SessionListResponse {
  sessions: SessionResponse[]
  total: number
  limit: number
  offset: number
}

export interface CreateSessionRequest {
  scenarioId: string
}

export interface JoinSessionRequest {
  inviteCode: string
}

export interface SessionPlayerResponse {
  id: string
  userId: string
  characterId: string | null
  status: string
  joinedAt: string
}

export interface SessionPlayerListResponse {
  players: SessionPlayerResponse[]
}

// --- Character types ---

export interface CharacterResponse {
  id: string
  userId: string
  name: string
  attributes: Record<string, unknown>
  inventory: unknown[]
  notes: string
  createdAt: string
  updatedAt: string
}

export interface CharacterListResponse {
  characters: CharacterResponse[]
  total: number
  limit: number
  offset: number
}

export interface CreateCharacterRequest {
  name: string
  attributes: Record<string, unknown>
  inventory: unknown[]
  notes: string
}

export interface UpdateCharacterRequest {
  name: string
  attributes: Record<string, unknown>
  inventory: unknown[]
  notes: string
}

export interface AssignCharacterRequest {
  characterId: string
}

// --- WebSocket types (snake_case JSON, matching Go realtime/message.go) ---

/** Server → Client envelope */
export interface WsEnvelope {
  type: string
  session_id: string
  sender_id: string
  target_ids?: string[]
  payload: unknown
  timestamp: number
}

/** Client → Server action */
export interface WsAction {
  type: string
  payload: unknown
}

// GM Action payloads
export interface AdvanceScenePayload {
  scene_id: string
}

export interface DiceRollPayload {
  formula: string
  purpose?: string
}

export interface RevealItemPayload {
  item_id: string
  player_ids?: string[]
}

export interface RevealNPCFieldPayload {
  npc_id: string
  field_key: string
  player_ids?: string[]
}

export interface GMBroadcastPayload {
  content?: string
  image_url?: string
  player_ids?: string[]
}

// Player Action payloads
export interface PlayerChoicePayload {
  transition_index: number
}

// --- GameState types (snake_case, matching Go realtime/gamestate.go) ---

export interface GameState {
  session_id: string
  status: string
  current_scene: string
  players: Record<string, PlayerState>
  dice_history: DiceResult[]
  variables: Record<string, unknown>
  revealed_items: Record<string, string[]>
  revealed_npc_fields: Record<string, Record<string, string[]>>
  last_sequence: number
}

export interface PlayerState {
  user_id: string
  current_scene: string
}

export interface DiceResult {
  formula: string
  results: number[]
  modifier: number
  total: number
}

// --- ScenarioContent types (snake_case, matching Go realtime/scenario.go) ---

export interface ScenarioContent {
  id: string
  title: string
  start_scene: string
  scenes: Scene[]
  items: Item[]
  npcs: NPC[]
  variables: ScenarioVariable[]
  rules?: Rules
}

export interface Scene {
  id: string
  name: string
  content: string
  gm_notes?: string
  items_available?: string[]
  npcs_present?: string[]
  transitions?: Transition[]
}

export interface Transition {
  target: string
  trigger: string
  condition?: string
  label?: string
}

export interface Item {
  id: string
  name: string
  type: string
  description: string
  image?: string
}

export interface NPC {
  id: string
  name: string
  image?: string
  fields?: NPCField[]
}

export interface NPCField {
  key: string
  label: string
  value: string
  visibility: string
}

export interface ScenarioVariable {
  name: string
  type: string
  default: unknown
}

export interface Rules {
  attributes?: Attribute[]
  dice_formula?: string
  check_method?: string
}

export interface Attribute {
  name: string
  display: string
  default: number
}

// --- Frontend-only types ---

export interface EventLogEntry {
  id: string
  type: string
  senderId: string
  payload: unknown
  timestamp: number
  sequence: number
}

export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting'
