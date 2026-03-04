# [ADR-002]: 即時通訊策略

| 欄位 | 內容 |
|------|------|
| **狀態** | `Accepted` |
| **日期** | 2026-02-27 |
| **決策者** | 專案擁有者 |

---

## 背景（Context）

TRPG-Simulation 是線上 TRPG 遊戲輔助平台，GM 和玩家需要即時同步：場景切換、道具揭露、骰子檢定、GM 投放訊息、玩家選擇都必須低延遲推送。語音與文字聊天由外部工具（如 Discord）處理，平台專注於遊戲機制的即時同步。需要決定即時通訊的技術方案和架構模式。

---

## 評估選項（Options Considered）

### 選項 A：REST + WebSocket 混合

- **優點**：各司其職。REST 處理 CRUD（認證、劇本管理、Session 建立）；WebSocket 處理遊戲內即時事件。保留 HTTP 標準工具鏈（快取、middleware、OpenAPI）
- **缺點**：前端需維護兩種連線
- **風險**：低。業界成熟模式

### 選項 B：純 WebSocket

- **優點**：統一通訊層
- **缺點**：需重新實作 request-response 模式（含錯誤處理、超時、重試）、失去 HTTP 快取和 OpenAPI 文件
- **風險**：中。過度設計，增加前後端複雜度

### 選項 C：Server-Sent Events (SSE) + REST

- **優點**：SSE 比 WebSocket 簡單、瀏覽器原生支援
- **缺點**：SSE 是單向（server → client），玩家操作仍需 REST；不適合雙向高頻互動
- **風險**：中。遊戲場景需要雙向即時，SSE 不足

---

## 決策（Decision）

選擇 **選項 A：REST + WebSocket 混合**。

### WebSocket 架構：Hub-Room 模式

```
Hub (管理所有 Room 的生命週期)
 ├── Room A (GameSession 1)
 │    ├── GM 連線 (完整可見性)
 │    ├── Player 1 連線 (過濾後資料)
 │    └── Player 2 連線 (過濾後資料)
 └── Room B (GameSession 2)
      ├── GM 連線
      └── Player 1 連線
```

- **Hub**：單一 goroutine，管理 Room 建立/銷毀、連線路由
- **Room**：每個 GameSession 一個 Room，獨立 goroutine 處理訊息廣播
- **Client**：每個 WebSocket 連線一個 Client 結構，包含讀/寫 goroutine

### 訊息信封格式

```json
{
  "type": "<event_type>",
  "session_id": "uuid",
  "sender_id": "uuid",
  "target_ids": ["uuid"],
  "payload": {},
  "timestamp": 1709020800
}
```

#### Server→Client 事件類型

| 事件 | 說明 |
|------|------|
| `game_started` | 遊戲開始 |
| `game_paused` | 遊戲暫停 |
| `game_resumed` | 遊戲恢復 |
| `game_ended` | 遊戲結束 |
| `state_sync` | 連線時全量狀態同步 |
| `scene_changed` | 場景切換 |
| `dice_rolled` | 骰子結果 |
| `item_revealed` | 道具揭露（legacy，向後相容） |
| `item_given` | 道具給予（背包系統） |
| `item_removed` | 道具移除（背包系統） |
| `npc_field_revealed` | NPC 欄位揭露 |
| `variable_changed` | 劇本變數變更 |
| `player_votes` | 轉場投票統計 |
| `gm_broadcast` | GM 投放訊息 |
| `transitions_updated` | 轉場條件重算結果（per-player） |
| `player_joined` | 玩家加入 |
| `player_left` | 玩家離開 |
| `error` | 錯誤回應 |

#### Client→Server 動作類型

| 動作 | 角色 | 說明 |
|------|------|------|
| `start_game` | GM | 開始遊戲 |
| `pause_game` | GM | 暫停遊戲 |
| `resume_game` | GM | 恢復遊戲 |
| `end_game` | GM | 結束遊戲 |
| `advance_scene` | GM | 切換場景 |
| `dice_roll` | GM/Player | 擲骰 |
| `reveal_item` | GM | 揭露道具（legacy） |
| `give_item` | GM | 給予道具到背包 |
| `remove_item` | GM | 移除背包道具 |
| `reveal_npc_field` | GM | 揭露 NPC 欄位 |
| `player_choice` | Player | 玩家場景選擇/投票 |
| `gm_broadcast` | GM | 投放訊息 |
| `set_variable` | GM | 修改劇本變數 |

### REST vs WebSocket 職責劃分

| 操作 | 協議 | 理由 |
|------|------|------|
| 用戶認證（登入/註冊） | REST | 標準 request-response |
| 劇本 CRUD | REST | 非即時操作 |
| GameSession 建立/列表 | REST | 非即時操作 |
| 加入遊戲（邀請碼） | REST | 一次性操作 |
| 場景切換 | WebSocket | 即時推送給所有玩家 |
| 道具揭露/給予/移除 | WebSocket | 即時推送 |
| NPC 角色卡欄位揭露 | WebSocket | 即時推送給指定玩家 |
| 骰子檢定 | WebSocket | 即時結果 |
| GM 投放訊息 | WebSocket | GM 即時推送文字/圖片給指定玩家 |
| 玩家選擇/投票 | WebSocket | 即時互動 |
| 劇本變數修改 | WebSocket | GM 即時修改遊戲狀態 |
| 上傳圖片 | REST | 非即時，multipart/form-data |
| GM/玩家筆記儲存 | REST | 非即時，個人備忘錄 |
| 遊戲狀態同步 | WebSocket | 斷線重連時全量同步 |

### 斷線重連機制

1. Client 維護 `lastEventId`
2. 斷線後重新建立 WebSocket 連線，連線時帶上 `lastEventId`
3. Server 從 `game_events` 表重放缺失事件
4. Heartbeat：Server 每 30 秒 ping，Client 10 秒內無 pong 則標記 disconnected

### Go 實作選擇

- WebSocket 庫：`github.com/gorilla/websocket`
- 每個 Client 2 個 goroutine（read pump + write pump）
- Room 使用 channel 做訊息路由，避免 mutex

---

## 後果（Consequences）

**正面影響：**
- REST 和 WebSocket 各司其職，降低複雜度
- Hub-Room 模式天然支援多遊戲同時進行
- 斷線重連基於 event replay，可靠且簡單

**負面影響 / 技術債：**
- 前端需維護 REST client + WebSocket client 兩套連線邏輯
- 單實例 Hub 限制橫向擴展（未來需 Redis pub/sub）

**後續追蹤：**
- [x] SPEC-005：WebSocket 連線生命週期與訊息類型定義（已完成）
- [x] SPEC-005：斷線重連 protocol（已完成，基於 `last_event_seq` 事件重放）

---

## 關聯（Relations）

- 取代：（無）
- 被取代：（無）
- 參考：ADR-001（技術棧選型）
