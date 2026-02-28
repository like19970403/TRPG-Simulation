# CLAUDE.md

本檔案為 Claude Code (claude.ai/code) 在此 repository 工作時的指引。

---

## 關於本 Repository

AI-SOP-Protocol (ASP) 是 AI 編程助手的行為憲法框架。將開發文化（ADR 先於實作、TDD、文件同步、破壞性操作防護）編碼為機器可讀的約束，讓 Claude Code 自動遵守。ASP 規範的是**怎麼做**（流程），不管**做什麼**（產品方向）。

**本 repo 就是 ASP 框架本身。** 這裡的檔案（CLAUDE.md、Makefile、profiles、hooks、install.sh）會被安裝到目標專案中。修改時請記住：你改的是會發佈給使用者的模板。

版本號位於 `.asp/VERSION`（semver）。Makefile 頂部有獨立版本標記 `ASP_MAKEFILE_VERSION`。

## 開發指令

本 repo 沒有自己的建置系統或測試套件。Makefile 是給目標專案用的**模板**（多語言：依序嘗試 Go、Python、Node）。

```bash
# 驗證 install 腳本語法
bash -n .asp/scripts/install.sh

# 驗證 hook 腳本語法
bash -n .asp/hooks/clean-allow-list.sh

# 在暫存目錄乾跑安裝
mkdir /tmp/asp-test && cd /tmp/asp-test && git init && bash /path/to/install.sh
```

Commit 遵循 conventional-commits：`feat:`、`fix:`、`refactor:`、`chore:`、`docs:`。

## 架構

### 分層 Profile 系統

Profile 是可組合的分層，由目標專案中的 `.ai_profile` YAML 選擇載入：

```
第 1 層：鐵則（CLAUDE.md）                    — 永遠載入，不可覆蓋
第 2 層：全域準則（global_core.md）             — 所有專案類型必載
第 3 層：專案類型（system_dev.md 或 content_creative.md）
第 4 層：作業模式（multi_agent.md 或 committee.md）— 可選
第 5 層：開發策略（vibe_coding.md）              — 可選
第 6 層：選配（rag_context.md、guardrail.md、coding_style.md、openapi.md、frontend_design.md）— 可選
```

Profile 對應由 `.ai_profile` 欄位驅動 → 見下方 Profile 對應表。

### 關鍵檔案

| 檔案 | 用途 |
|------|------|
| `.asp/scripts/install.sh` | 一鍵安裝腳本（457 行）。處理全新安裝、升級、舊版遷移、settings.json 合併、.gitignore 合併。支援非互動模式（環境變數 `ASP_TYPE`、`ASP_NAME`、`ASP_RAG`、`ASP_GUARDRAIL`、`ASP_HITL`）。 |
| `.asp/hooks/clean-allow-list.sh` | SessionStart hook。用 `jq` 從 `.claude/settings.local.json` 移除危險 Bash allow 規則。匹配模式：`git rebase/push`、`docker push/deploy`、`rm -r*`、`find -delete`。 |
| `.asp/profiles/` | 11 個 profile 檔，使用混合表達：自然語言（哲學）、pseudocode（`FUNCTION/IF/MATCH/INVARIANT` 決策邏輯）、bash/make（技術執行）、表格/YAML（靜態規則）。 |
| `.asp/templates/` | ADR、SPEC、架構模板 + 預設 `.ai_profile`（`.system`、`.content`、`.full`）。 |
| `.asp/scripts/rag/` | 可選 RAG 支援：ChromaDB + sentence-transformers 索引建立、搜尋、統計。 |
| `Makefile` | 發佈到目標專案的模板 Makefile，版本與 ASP 版本獨立管理。 |

### 修改時須遵守的設計原則

- **鐵則只有 3 條**——技術強制，永遠不能多加。「一條有條件的規則，勝過三條無條件的規則。」
- **預設行為可跳過但須說明理由**——教 AI 學會判斷，而非只是服從。
- **Token 經濟**——shell 指令超過 3 行就移入 Makefile；RAG 模式存在是為了避免把所有 profile 塞進 context；content 類型專案跳過所有 Docker/TDD/CI 邏輯。
- **SessionStart hook + 內建權限**（v1.3+ 作法）取代 PreToolUse hooks（v1.1-v1.2）。更簡單、更可靠。
- **install.sh 必須防禦性編寫**——處理既有 CLAUDE.md、既有 Makefile（保留 APP_NAME）、既有 .ai_profile（只補缺漏欄位）、舊版目錄清理。

---

# AI-SOP-Protocol (ASP) — 行為憲法

> 以下是 ASP 核心協議，會被安裝到目標專案中。
> 讀取順序：本檔案 → `.ai_profile` → 對應 `.asp/profiles/`（按需）

---

## 啟動程序

1. 讀取 `.ai_profile`，依欄位載入對應 profile
2. **RAG 已啟用時**：回答任何專案架構/規格問題前，先執行 `make rag-search Q="..."`
3. 無 `.ai_profile` 時：只套用本檔案鐵則，詢問使用者專案類型

```yaml
# .ai_profile 完整欄位參考
type:      system | content | architecture   # 必填
mode:      single | multi-agent | committee  # 預設 single
workflow:  standard | vibe-coding            # 預設 standard
rag:       enabled | disabled               # 預設 disabled
guardrail:    enabled | disabled               # 預設 disabled
coding_style: enabled | disabled               # 預設 disabled
openapi:      enabled | disabled               # 預設 disabled
frontend_design: enabled | disabled            # 預設 disabled
hitl:         minimal | standard | strict      # 預設 standard
name:         your-project-name
```

**Profile 對應表：**

| 欄位值 | 載入的 Profile |
|--------|----------------|
| `type: system` | `.asp/profiles/global_core.md` + `.asp/profiles/system_dev.md` |
| `type: content` | `.asp/profiles/global_core.md` + `.asp/profiles/content_creative.md` |
| `type: architecture` | `.asp/profiles/global_core.md` + `.asp/profiles/system_dev.md` |
| `mode: multi-agent` | + `.asp/profiles/multi_agent.md` |
| `mode: committee` | + `.asp/profiles/committee.md` |
| `workflow: vibe-coding` | + `.asp/profiles/vibe_coding.md` |
| `rag: enabled` | + `.asp/profiles/rag_context.md` |
| `guardrail: enabled` | + `.asp/profiles/guardrail.md` |
| `coding_style: enabled` | + `.asp/profiles/coding_style.md` |
| `openapi: enabled` | + `.asp/profiles/openapi.md` |
| `frontend_design: enabled` | + `.asp/profiles/frontend_design.md` |

---

## 🔴 鐵則（不可覆蓋）

以下規則在任何情況下不得繞過：

| 鐵則 | 說明 |
|------|------|
| **破壞性操作防護** | `rebase / rm -rf / docker push / git push` 等危險操作由 Claude Code 內建權限系統確認（SessionStart hook 自動清理 allow list）；`git push` 前必須先列出變更摘要並等待人類明確同意 |
| **敏感資訊保護** | 禁止輸出任何 API Key、密碼、憑證，無論何種包裝方式 |
| **ADR 未定案禁止實作** | ADR 狀態為 Draft 時，禁止撰寫對應的生產代碼；必須等 ADR 進入 Accepted 狀態 |

---

## 🟡 預設行為（有充分理由可調整，但必須說明）

| 預設行為 | 可跳過的條件 |
|----------|-------------|
| ADR 優先於實作 | 修改範圍僅限單一函數，且無架構影響 |
| TDD：新功能必須測試先於代碼 | Bug 修復和原型驗證可跳過，需標記 `tech-debt: test-pending` |
| 非 trivial 修改需先建 SPEC | trivial（單行/typo/配置）可豁免，需說明理由 |
| 文件同步更新 | 緊急修復可延後，但同一 session 結束前必須補齊文件 |
| Bug 修復後 grep 全專案 | 所有 Bug 修復後一律 grep，無豁免 |
| Makefile 優先 | 緊急修復或 make 目標不存在時，可直接執行原生指令，需說明理由 |

---

## 標準工作流

```
需求 → [ADR 建立] → [UI 設計] → SDD 設計 → TDD 測試 → 實作 → 文件同步 → 確認後部署
         ↑ 架構影響時必須  ↑ frontend_design: enabled 時   ↑ 預設行為，可調整
```

---

## Makefile 速查

| 動作 | 指令 |
|------|------|
| 建立 Image | `make build` |
| 清理環境 | `make clean` |
| 重新部署 | `make deploy` |
| 執行測試 | `make test` |
| 局部測試 | `make test-filter FILTER=xxx` |
| 新增 ADR | `make adr-new TITLE="..."` |
| 新增規格書 | `make spec-new TITLE="..."` |
| 查詢知識庫 | `make rag-search Q="..."` |
| Agent 完成回報 | `make agent-done TASK=xxx STATUS=success` |
| 儲存 Session | `make session-checkpoint NEXT="..."` |

> 以上為常用指令，完整列表請執行 `make help`

---

## 技術執行層（Hooks + 內建權限）

ASP 使用 Claude Code 內建權限系統 + SessionStart Hook 保護危險操作：

| 機制 | 說明 |
|------|------|
| **內建權限系統** | 危險指令（git push/rebase, docker push, rm -rf 等）不在 allow list 中時，Claude Code 自動彈出「Allow this bash command?」確認框 |
| **SessionStart Hook** | `clean-allow-list.sh` 每次 session 啟動時自動清理 allow list 中的危險規則，確保內建權限系統持續生效 |

> 設定檔位於 `.claude/settings.json`，hook 腳本位於 `.asp/hooks/`。
> 使用者可在確認框中選擇 "Allow"（一次性）或 "Always allow"（永久），但後者會在下次 session 啟動時被自動清理。
