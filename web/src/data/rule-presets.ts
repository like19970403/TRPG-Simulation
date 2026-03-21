import type { Rules, Item, ScenarioVariable } from '../api/types'

export interface ProfileField {
  key: string
  label: string
  type: 'text' | 'textarea'
  placeholder: string
}

export interface SkillDefinition {
  id: string
  name: string
  level: '初級' | '中級' | '高級'
  cost?: number
  effect: string
  special?: string
  attribute?: string
  weaponType?: 'palm' | 'blade' | 'spear' | 'sword' | 'hidden'
}

export interface RulePreset {
  id: 'wuxia' | 'detective'
  name: string
  description: string
  rules: Rules
  suggestedVariables: ScenarioVariable[]
  suggestedItems: Item[]
  profileFields: ProfileField[]
  attributeDescriptions: Record<string, string>
  martialSkills?: SkillDefinition[]
  cultivationMethods?: SkillDefinition[]
  startingSkillSlots?: number
  startingCultivationSlots?: number
  startingWeapons?: Item[]
}

export const RULE_PRESETS: RulePreset[] = [
  {
    id: 'wuxia',
    name: '武俠（江湖風雲錄）',
    description: '輕功、劍法、內力，闖蕩江湖的武俠冒險',
    rules: {
      attributes: [
        { name: 'martial', display: '武功', default: 5 },
        { name: 'inner_force', display: '內力', default: 5 },
        { name: 'agility', display: '身法', default: 5 },
        { name: 'wisdom', display: '機智', default: 5 },
      ],
      dice_formula: '2d6',
      check_method: 'gte',
      gm_reference: `## 江湖風雲錄 — GM 快速參考

### 檢定方式
擲 **2d6 + 屬性值** ≥ 目標值即成功。

### 難度表

| 難度 | 目標值 | 範例 |
|------|--------|------|
| 容易 | 8 | 翻牆、辨認草藥 |
| 普通 | 10 | 擋住山賊攻擊、飛簷走壁 |
| 困難 | 12 | 對戰高手、解奇毒 |
| 傳奇 | 14 | 挑戰武林盟主 |

### 四大屬性
- **武功** — 劍法、拳腳、兵器技巧
- **內力** — 內功修為、療傷、抗毒
- **身法** — 輕功、閃避、潛行
- **機智** — 謀略、話術、江湖見識

### 特殊機制

**燃內力**：消耗 1 個「內力點」道具，該次檢定 **+2**。內力點用盡有走火入魔風險（敘事處理）。

**對決**：雙方各擲 2d6+武功，高者勝，三局兩勝定勝負。

**秘笈**：取得武功秘笈（key_item）後，對應屬性永久 +1。

**江湖聲望**：透過「reputation」變數追蹤。正值=俠義、負值=惡名，可作為場景轉場條件。

### 裝備欄位
- **雙手**：左手 + 右手各可裝一件武器（雙手武器如槍佔兩格）
- **防具**：頭、身、褲、鞋各一件
- 武學必須裝備對應武器才能施展（掌法空手可用）

### 武學系統（五大兵器）
施展武學 → GM 用 remove_item 扣內力點 → 擲骰加技能加成。
戰鬥結束後，give_item 補回內力點（初始 3 個）。

| 武學 | 兵器 | 消耗 | 效果 |
|------|------|------|------|
| 鐵砂掌 | 掌 | 2 | 武功 +2，可繳械 |
| 疾風刀法 | 刀 | 2 | 武功 +3（純爆發） |
| 盤龍槍法 | 槍 | 2 | 武功 +2，獲得先手 |
| 流雲劍法 | 劍 | 2 | 武功 +2，被攻擊時防禦 +2 |
| 梅花針 | 暗器 | 1 | 機智檢定，命中目標 -2 |

### 心法系統
一次只能運行一個，切換需在非戰鬥狀態。心法特效僅在使用對應武器/武學時觸發。

| 心法 | 對應 | 被動 | 特殊 |
|------|------|------|------|
| 金剛掌心法 | 掌 | 武功 +1 | 擒拿/控制 +1 |
| 烈刃心法 | 刀 | 武功 +1 | 刀法消耗 -1 |
| 定軍心法 | 槍 | 身法 +1 | 對方近身 -1 |
| 明鏡劍心 | 劍 | 機智 +1 | 可反擊 |
| 藏風心法 | 暗器 | 身法 +1 | 首次暗器突襲 +2 |

---

### 戰鬥流程（回合制）

**開始**：依身法排序（高→低），同值擲 1d6。
**每回合**：按順序行動，每人一次。

行動選項：攻擊（普攻/武學）、防禦（def×2）、使用道具、切換心法（耗行動）

### 傷害公式

\`\`\`
攻擊值 = 2d6 + 武功 + 武器atk + 武學加成 + 心法加成
防禦值 = 武功÷2 + 裝備def總和
防禦姿態 = 武功÷2 + 裝備def × 2
傷害 = max(0, 攻擊值 - 防禦值)
\`\`\`

**暗器**：攻擊用機智代替武功，防禦用身法÷2
**HP** = 10 + 內力 × 2

### 範例
攻：2d6(7) + 武功(5) + 鋼刀(2) + 疾風刀法(+3) + 烈刃心法(+1) = **18**
防：武功(5)÷2=2 + 竹笠(1)+布甲(2)+皮褲(1)+輕靴(1)=5 → **7**
傷害 = 18 - 7 = **11**`,
    },
    martialSkills: [
      { id: 'iron_palm', name: '鐵砂掌', level: '初級', cost: 2, effect: '武功 +2，成功時可繳械對方（GM 判定）', attribute: '武功', weaponType: 'palm' },
      { id: 'swift_blade', name: '疾風刀法', level: '初級', cost: 2, effect: '武功 +3', attribute: '武功', weaponType: 'blade' },
      { id: 'dragon_spear', name: '盤龍槍法', level: '初級', cost: 2, effect: '武功 +2，本回合獲得先手', attribute: '武功', weaponType: 'spear' },
      { id: 'flowing_sword', name: '流雲劍法', level: '初級', cost: 2, effect: '武功 +2，若被攻擊則防禦 +2', attribute: '武功', weaponType: 'sword' },
      { id: 'poison_needle', name: '梅花針', level: '初級', cost: 1, effect: '機智檢定攻擊，命中則目標下次檢定 -2', attribute: '機智', weaponType: 'hidden' },
    ],
    cultivationMethods: [
      { id: 'vajra_palm', name: '金剛掌心法', level: '初級', effect: '武功 +1', special: '擒拿/控制類檢定額外 +1' },
      { id: 'fierce_blade', name: '烈刃心法', level: '初級', effect: '武功 +1', special: '使用刀法武學時消耗 -1 內力（最低 1）' },
      { id: 'steady_spear', name: '定軍心法', level: '初級', effect: '身法 +1', special: '持槍時對方近身攻擊 -1' },
      { id: 'mirror_sword', name: '明鏡劍心', level: '初級', effect: '機智 +1', special: '被攻擊時可嘗試反擊' },
      { id: 'hidden_wind', name: '藏風心法', level: '初級', effect: '身法 +1', special: '首次暗器攻擊獲得突襲 +2' },
    ],
    startingSkillSlots: 2,
    startingCultivationSlots: 1,
    startingWeapons: [
      { id: 'iron_fist', name: '鐵拳套', type: 'weapon', description: '掌法武器，攻擊力 +1', slot: 'weapon', weapon_type: 'palm', atk: 1 },
      { id: 'steel_blade', name: '鋼刀', type: 'weapon', description: '刀法武器，攻擊力 +2', slot: 'weapon', weapon_type: 'blade', atk: 2 },
      { id: 'dragon_spear_weapon', name: '青龍槍', type: 'weapon', description: '槍法武器（雙手），攻擊力 +2', slot: 'weapon', weapon_type: 'spear', two_handed: true, atk: 2 },
      { id: 'green_sword', name: '青鋒劍', type: 'weapon', description: '劍法武器，攻擊力 +2', slot: 'weapon', weapon_type: 'sword', atk: 2 },
      { id: 'hidden_pouch', name: '暗器袋', type: 'weapon', description: '暗器武器，攻擊力 +1', slot: 'weapon', weapon_type: 'hidden', atk: 1 },
    ],
    attributeDescriptions: {
      '武功': '劍法、拳腳、兵器技巧',
      '內力': '內功修為、療傷、抗毒',
      '身法': '輕功、閃避、潛行',
      '機智': '謀略、話術、江湖見識',
    },
    profileFields: [
      { key: 'title', label: '外號', type: 'text', placeholder: '例：飛天蝠王、鐵掌水上飄' },
      { key: 'sect', label: '門派/出身', type: 'text', placeholder: '例：少林派、丐幫、江湖散人' },
      { key: 'appearance', label: '外貌', type: 'textarea', placeholder: '例：身材修長，眉目清秀，左臉頰有一道淡淡的劍傷' },
      { key: 'personality', label: '性格', type: 'textarea', placeholder: '例：表面冷漠寡言，實則重情重義，對不公之事絕不袖手旁觀' },
      { key: 'fighting_style', label: '武功路線', type: 'text', placeholder: '例：以柔克剛的內家拳法、暗器百發百中' },
      { key: 'backstory', label: '江湖經歷', type: 'textarea', placeholder: '例：十歲因滅門之仇入少林，苦修十年後下山尋仇…' },
      { key: 'goal', label: '目標/野心', type: 'textarea', placeholder: '例：找到殺害師父的兇手，為師門討回公道' },
    ],
    suggestedVariables: [
      { name: 'reputation', type: 'int', default: 0 },
      { name: 'boss_defeated', type: 'bool', default: false },
      { name: 'hp_player1', type: 'int', default: 20 },
      { name: 'hp_player2', type: 'int', default: 20 },
      { name: 'hp_player3', type: 'int', default: 20 },
      { name: 'hp_player4', type: 'int', default: 20 },
      { name: 'combat_status', type: 'string', default: '' },
      { name: 'combat_enemy_name', type: 'string', default: '' },
      { name: 'combat_enemy_hp', type: 'int', default: 0 },
      { name: 'combat_enemy_max_hp', type: 'int', default: 0 },
      { name: 'combat_enemy_martial', type: 'int', default: 5 },
      { name: 'combat_enemy_def', type: 'int', default: 0 },
      { name: 'combat_enemy_agility', type: 'int', default: 5 },
      { name: 'combat_enemy_weapon_atk', type: 'int', default: 2 },
      { name: 'combat_round', type: 'int', default: 0 },
    ],
    suggestedItems: [
      {
        id: 'inner_force_point',
        name: '內力點',
        type: 'consumable',
        description:
          '你的內力儲備。燃燒一點可在檢定時 +2，但用盡將有走火入魔的危險。',
        stackable: true,
      },
      {
        id: 'antidote',
        name: '解毒丹',
        type: 'consumable',
        description: '武林中流通的萬用解毒藥丸。',
        stackable: true,
      },
      {
        id: 'secret_manual',
        name: '武功秘笈',
        type: 'key_item',
        description: '傳說中的武功秘笈，習得後可永久提升一項屬性。',
        gm_notes: '給予後記得該玩家對應屬性檢定永久 +1',
      },
      // 武器
      { id: 'iron_fist', name: '鐵拳套', type: 'weapon', description: '包裹鐵片的拳套，掌法專用。攻擊力 +1。', slot: 'weapon', weapon_type: 'palm', atk: 1 },
      { id: 'steel_blade', name: '鋼刀', type: 'weapon', description: '鋒利的單刃鋼刀。攻擊力 +2。', slot: 'weapon', weapon_type: 'blade', atk: 2 },
      { id: 'dragon_spear_weapon', name: '青龍槍', type: 'weapon', description: '長柄大槍，需雙手持握。攻擊力 +2。', slot: 'weapon', weapon_type: 'spear', two_handed: true, atk: 2 },
      { id: 'green_sword', name: '青鋒劍', type: 'weapon', description: '輕盈鋒利的長劍。攻擊力 +2。', slot: 'weapon', weapon_type: 'sword', atk: 2 },
      { id: 'hidden_pouch', name: '暗器袋', type: 'weapon', description: '裝滿各式暗器的皮袋。攻擊力 +1。', slot: 'weapon', weapon_type: 'hidden', atk: 1 },
      // 武學 items（掌、刀、槍、劍、暗器）
      { id: 'iron_palm', name: '鐵砂掌', type: 'martial_skill', description: '掌法｜消耗 2 內力，武功 +2，成功時可繳械對方（GM 判定）。', gm_notes: '消耗: 2 內力點\n效果: 武功 +2\n特殊: 成功可繳械' },
      { id: 'swift_blade', name: '疾風刀法', type: 'martial_skill', description: '刀法｜消耗 2 內力，武功 +3。純粹的爆發傷害。', gm_notes: '消耗: 2 內力點\n效果: 武功 +3' },
      { id: 'dragon_spear', name: '盤龍槍法', type: 'martial_skill', description: '槍法｜消耗 2 內力，武功 +2，本回合獲得先手。', gm_notes: '消耗: 2 內力點\n效果: 武功 +2\n特殊: 先手行動' },
      { id: 'flowing_sword', name: '流雲劍法', type: 'martial_skill', description: '劍法｜消耗 2 內力，武功 +2，若對方本回合也攻擊你，防禦 +2。', gm_notes: '消耗: 2 內力點\n效果: 武功 +2\n特殊: 被攻擊時防禦 +2' },
      { id: 'poison_needle', name: '梅花針', type: 'martial_skill', description: '暗器｜消耗 1 內力，以機智檢定攻擊，命中則目標下次檢定 -2。', gm_notes: '消耗: 1 內力點\n效果: 機智檢定攻擊\n特殊: 命中目標下次 -2' },
      // 心法 items（對應五大兵器路線）
      { id: 'vajra_palm', name: '金剛掌心法', type: 'cultivation_method', description: '掌法心法｜武功 +1，擒拿/控制類檢定額外 +1。', gm_notes: '被動: 武功 +1\n特殊: 擒拿/控制 +1' },
      { id: 'fierce_blade', name: '烈刃心法', type: 'cultivation_method', description: '刀法心法｜武功 +1，使用刀法武學時消耗 -1 內力（最低 1）。', gm_notes: '被動: 武功 +1\n特殊: 刀法消耗 -1' },
      { id: 'steady_spear', name: '定軍心法', type: 'cultivation_method', description: '槍法心法｜身法 +1，持槍時對方近身攻擊 -1。', gm_notes: '被動: 身法 +1\n特殊: 對方近身攻擊 -1' },
      { id: 'mirror_sword', name: '明鏡劍心', type: 'cultivation_method', description: '劍法心法｜機智 +1，被攻擊時可嘗試反擊（額外攻擊機會）。', gm_notes: '被動: 機智 +1\n特殊: 可反擊' },
      { id: 'hidden_wind', name: '藏風心法', type: 'cultivation_method', description: '暗器心法｜身法 +1，首次暗器攻擊自動獲得突襲 +2。', gm_notes: '被動: 身法 +1\n特殊: 首次暗器突襲 +2' },
    ],
  },
  {
    id: 'detective',
    name: '偵探推理（迷霧真相）',
    description: '搜證、審問、推理，揭開真相的推理冒險',
    rules: {
      attributes: [
        { name: 'observe', display: '觀察', default: 5 },
        { name: 'reason', display: '推理', default: 5 },
        { name: 'social', display: '交際', default: 5 },
        { name: 'nerve', display: '膽識', default: 5 },
      ],
      dice_formula: '2d6',
      check_method: 'gte',
      gm_reference: `## 迷霧真相 — GM 快速參考

### 檢定方式
擲 **2d6 + 屬性值** ≥ 目標值即成功。

### 難度表

| 難度 | 目標值 | 範例 |
|------|--------|------|
| 容易 | 8 | 搜查顯眼的房間 |
| 普通 | 10 | 發現隱藏的線索、說服證人 |
| 困難 | 12 | 看穿精心偽裝、審問老練嫌疑人 |
| 傳奇 | 14 | 破解完美犯罪的關鍵 |

### 四大屬性
- **觀察** — 搜證、鑑識、注意細節
- **推理** — 邏輯分析、連結線索、還原事件
- **交際** — 審問、說服、讀取肢體語言
- **膽識** — 臨危不亂、面對危險、抵抗威脅

### 特殊機制

**線索品質**：關鍵線索必定找到（不擲骰）。骰子決定額外資訊深度：
- 失敗 → 只得到基本資訊
- 成功 → 額外細節（揭露隱藏 NPC 欄位）
- 大成功 (超過目標值 4+) → 關鍵洞見

**壓力點**：消耗 1 個「壓力點」道具，審問時可重骰或 **+2**。用盡後嫌疑人拒絕配合。

**線索計數**：透過「clue_count」變數追蹤已發現線索數，達到門檻可解鎖「推理總結」場景。

**嫌疑值**：為每位嫌疑人設定獨立變數，累積到門檻可解鎖對質場景。`,
    },
    attributeDescriptions: {
      '觀察': '搜證、鑑識、注意細節',
      '推理': '邏輯分析、連結線索、還原事件',
      '交際': '審問、說服、讀取肢體語言',
      '膽識': '臨危不亂、面對危險、抵抗威脅',
    },
    profileFields: [
      { key: 'profession', label: '職業', type: 'text', placeholder: '例：退休刑警、法醫助理、自由記者' },
      { key: 'appearance', label: '外貌', type: 'textarea', placeholder: '例：四十出頭，總是穿著皺巴巴的風衣，右手無名指少了一截' },
      { key: 'personality', label: '性格', type: 'textarea', placeholder: '例：直覺敏銳但脾氣急躁，對細節有近乎偏執的要求' },
      { key: 'specialty', label: '特長', type: 'text', placeholder: '例：指紋鑑識專家、精通犯罪心理學' },
      { key: 'past_cases', label: '過往案件', type: 'textarea', placeholder: '例：三年前偵破了震驚社會的連環失蹤案，但真相讓他至今無法釋懷…' },
      { key: 'weakness', label: '弱點/陰影', type: 'textarea', placeholder: '例：酗酒、失眠、對黑暗封閉空間有嚴重恐懼' },
      { key: 'goal', label: '目標/動機', type: 'textarea', placeholder: '例：查明搭檔五年前殉職的真正原因' },
    ],
    suggestedVariables: [
      { name: 'clue_count', type: 'int', default: 0 },
      { name: 'case_solved', type: 'bool', default: false },
    ],
    suggestedItems: [
      {
        id: 'pressure_point',
        name: '壓力點',
        type: 'consumable',
        description:
          '你的精力和專注力。消耗一點可在審問時重骰或 +2，用盡後嫌疑人將拒絕配合。',
        stackable: true,
      },
      {
        id: 'magnifying_glass',
        name: '放大鏡',
        type: 'key_item',
        description: '精密的放大鏡，有助於觀察微小證據。',
        gm_notes: '給予後觀察檢定永久 +1',
      },
      {
        id: 'notebook',
        name: '偵探筆記本',
        type: 'key_item',
        description: '你的隨身筆記本，記錄了所有發現和推理。',
      },
    ],
  },
]
