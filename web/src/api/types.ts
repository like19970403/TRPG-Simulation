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

export interface GiveItemPayload {
  item_id: string
  player_id?: string
  player_ids?: string[]
  quantity?: number
}

export interface RemoveItemPayload {
  item_id: string
  player_id?: string
  player_ids?: string[]
  quantity?: number
}

export interface GMBroadcastPayload {
  content?: string
  image_url?: string
  player_ids?: string[]
}

// GM set variable payload
export interface SetVariablePayload {
  name: string
  value: unknown
}

// Player Action payloads
export interface PlayerChoicePayload {
  transition_index: number
}

// Client → Server action payload mapping (type-safe sendAction)
export interface ActionPayloadMap {
  start_game: Record<string, never>
  pause_game: { reason?: string }
  resume_game: Record<string, never>
  end_game: { reason?: string }
  advance_scene: AdvanceScenePayload
  dice_roll: DiceRollPayload
  reveal_item: RevealItemPayload
  give_item: GiveItemPayload
  remove_item: RemoveItemPayload
  reveal_npc_field: RevealNPCFieldPayload
  player_choice: PlayerChoicePayload
  gm_broadcast: GMBroadcastPayload
  set_variable: SetVariablePayload
}

export type ActionType = keyof ActionPayloadMap

// Vote tally from server (player_votes event)
export interface VoteTallyEntry {
  count: number
  voters: string[]
}

// --- GameState types (snake_case, matching Go realtime/gamestate.go) ---

export interface InventoryEntry {
  item_id: string
  quantity: number
}

export interface GameState {
  session_id: string
  status: string
  current_scene: string
  players: Record<string, PlayerState>
  player_attributes: Record<string, Record<string, unknown>>
  dice_history: DiceResult[]
  variables: Record<string, unknown>
  revealed_items: Record<string, string[]>
  revealed_npc_fields: Record<string, Record<string, string[]>>
  player_inventory: Record<string, InventoryEntry[]>
  last_sequence: number
}

export interface PlayerState {
  user_id: string
  username: string
  character_id?: string
  character_name?: string
  current_scene: string
  online: boolean
}

export interface DiceResult {
  roller_id?: string
  roller_name?: string
  formula: string
  results: number[]
  modifier: number
  total: number
  purpose?: string
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
  on_enter?: Action[]
  on_exit?: Action[]
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
  gm_notes?: string
  image?: string
  stackable?: boolean
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

// --- Action types (matching Go realtime/scenario.go) ---

export interface Action {
  set_var?: SetVarAction
  reveal_item?: RevealItemAction
  give_item?: GiveItemAction
  remove_item?: RemoveItemAction
  reveal_npc_field?: RevealNPCFieldAction
}

export interface GiveItemAction {
  item_id: string
  to: string
  quantity?: number
}

export interface RemoveItemAction {
  item_id: string
  from: string
  quantity?: number
}

export interface SetVarAction {
  name: string
  value: unknown
  expr?: string
}

export interface RevealItemAction {
  item_id: string
  to: string
}

export interface RevealNPCFieldAction {
  npc_id: string
  field_key: string
  to: string
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

// --- Image upload types ---

export interface ImageUploadResponse {
  url: string
  filename: string
}

// --- Replay types ---

export interface ReplayEvent {
  id: string
  sequence: number
  type: string
  actorId?: string
  payload: unknown
  createdAt: string
}
