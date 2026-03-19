# Scenario Content JSON Format Guide

This guide explains every field in the Scenario Content JSON format used by the TRPG Simulation platform.

For a complete working example, see [`sample-scenario.json`](./sample-scenario.json).

---

## Top-Level Structure

```json
{
  "id": "string",
  "title": "string",
  "start_scene": "string (scene ID)",
  "scenes": [],
  "items": [],
  "npcs": [],
  "variables": [],
  "rules": {}
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier for the scenario |
| `title` | string | Yes | Display name of the scenario |
| `start_scene` | string | **Yes** | ID of the first scene when the game starts |
| `scenes` | Scene[] | **Yes** (≥1) | Array of scenes in the scenario |
| `items` | Item[] | No | Array of items/clues available in the scenario |
| `npcs` | NPC[] | No | Array of non-player characters |
| `variables` | Variable[] | No | Array of scenario-level variables |
| `rules` | Rules | No | Custom rules (attributes, dice, checks) |

---

## Scene

Each scene represents a location, event, or moment in the story.

```json
{
  "id": "library",
  "name": "The Library",
  "content": "Markdown text describing the scene...",
  "gm_notes": "Private notes for the GM",
  "items_available": ["old_diary", "rusty_key"],
  "npcs_present": ["ghost"],
  "transitions": [],
  "on_enter": [],
  "on_exit": []
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique scene identifier (used in transitions) |
| `name` | string | Yes | Display name shown to players |
| `content` | string | Yes | Scene description in Markdown. Players see this. |
| `gm_notes` | string | No | Private notes visible only to the GM |
| `items_available` | string[] | No | Item IDs that can be found/revealed in this scene |
| `npcs_present` | string[] | No | NPC IDs present in this scene |
| `transitions` | Transition[] | No | Possible paths to other scenes |
| `on_enter` | Action[] | No | Actions executed when players enter this scene |
| `on_exit` | Action[] | No | Actions executed when players leave this scene |

---

## Transition

Transitions define the edges of the scene graph — how players move between scenes.

```json
{
  "target": "secret_room",
  "trigger": "player_choice",
  "condition": "has_key == true",
  "label": "Enter the Secret Room"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `target` | string | Yes | ID of the destination scene |
| `trigger` | string | Yes | `"player_choice"` or `"gm_decision"` |
| `condition` | string | No | Expression that must be true (uses `expr` engine) |
| `label` | string | No | Button/choice text shown to players or GM |

### Trigger Types

- **`player_choice`** — Appears as a clickable choice for the player. If `condition` is set, the choice is only available when the condition evaluates to `true`.
- **`gm_decision`** — Only visible to the GM in the console. The GM clicks to advance the scene.

### Condition Expressions

Conditions use the [`expr-lang/expr`](https://github.com/expr-lang/expr) engine. Variables defined in `variables` are available in expressions. Built-in functions are also available.

**Variable examples:**
- `has_key == true` — Boolean check
- `courage >= 5` — Numeric comparison
- `visited_library == true && has_key == true` — Compound condition

**Built-in functions:**
- `has_item("item_id")` — Returns `true` if the current player has the item in inventory
- `all_have_item("item_id")` — Returns `true` if all connected players have the item
- `item_count("item_id")` — Returns the quantity of the item in the current player's inventory
- `player_count()` — Returns the number of connected players

**Function examples:**
- `has_item("rusty_key")` — Check if the player has the key
- `item_count("gold_coin") >= 5` — Check if the player has at least 5 gold coins

---

## Item

Items represent objects, clues, or consumables that can be given to players' inventory.

```json
{
  "id": "rusty_key",
  "name": "Rusty Key",
  "type": "key_item",
  "description": "An old iron key covered in rust.",
  "gm_notes": "Required to enter the secret room. Consumed on use.",
  "image": "https://example.com/key.png",
  "stackable": false
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique item identifier |
| `name` | string | Yes | Display name |
| `type` | string | Yes | Category: `"key_item"`, `"clue"`, `"consumable"`, `"treasure"`, etc. |
| `description` | string | Yes | Description shown to players |
| `gm_notes` | string | No | Private notes visible only to the GM |
| `image` | string | No | Optional image URL |
| `stackable` | boolean | No | If `true`, quantity can exceed 1 (default: `false`) |

Items are managed through the **inventory system**:
1. `on_enter` / `on_exit` actions (`give_item`, `remove_item`) automatically manage inventory
2. GM can manually give or remove items from the console
3. Each player has an independent inventory (背包)
4. Non-stackable items cannot be given twice to the same player

---

## NPC

NPCs have public and hidden fields. The GM controls which fields players can see.

```json
{
  "id": "butler",
  "name": "The Old Butler",
  "image": "https://example.com/butler.png",
  "fields": [
    {
      "key": "appearance",
      "label": "Appearance",
      "value": "A tall, gaunt man in a tattered suit.",
      "visibility": "public"
    },
    {
      "key": "secret",
      "label": "Secret",
      "value": "He is actually a ghost.",
      "visibility": "hidden"
    },
    {
      "key": "weakness",
      "label": "Weakness",
      "value": "Show him his own portrait.",
      "visibility": "gm_only"
    }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique NPC identifier |
| `name` | string | Yes | Display name |
| `image` | string | No | Optional portrait URL |
| `fields` | NPCField[] | No | Array of information fields |

### NPC Field

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Yes | Unique field identifier within this NPC |
| `label` | string | Yes | Display label |
| `value` | string | Yes | The field content |
| `visibility` | string | Yes | `"public"`, `"hidden"`, or `"gm_only"` |

### Visibility Levels

- **`public`** — Visible to all players from the start
- **`hidden`** — Hidden by default, can be revealed via `on_enter` actions or GM manual reveal
- **`gm_only`** — Only visible to the GM, never revealed to players

---

## Variable

Variables track game state across scenes.

```json
{ "name": "has_key", "type": "bool", "default": false }
{ "name": "courage", "type": "int", "default": 0 }
{ "name": "player_name", "type": "string", "default": "" }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Variable name (used in conditions and set_var) |
| `type` | string | Yes | `"bool"`, `"int"`, or `"string"` |
| `default` | any | Yes | Initial value when the game starts |

---

## Actions (on_enter / on_exit)

Actions are executed automatically when players enter or leave a scene. Each action object has exactly one of the following fields:

### set_var

Sets a variable to a value.

```json
{ "set_var": { "name": "visited_library", "value": true } }
```

With expression evaluation:

```json
{ "set_var": { "name": "courage", "value": null, "expr": "courage + 2" } }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Variable name to set |
| `value` | any | Yes | Literal value (used if `expr` is empty) |
| `expr` | string | No | Expression to evaluate (result becomes the value) |

### give_item

Gives an item to a player's inventory.

```json
{ "give_item": { "item_id": "rusty_key", "to": "current_player" } }
{ "give_item": { "item_id": "gold_coin", "to": "all", "quantity": 3 } }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `item_id` | string | Yes | ID of the item to give |
| `to` | string | Yes | `"current_player"`, `"all"`, or a specific player ID |
| `quantity` | number | No | Number of items to give (default: `1`). Only meaningful for stackable items. |

### remove_item

Removes an item from a player's inventory.

```json
{ "remove_item": { "item_id": "rusty_key", "from": "current_player" } }
{ "remove_item": { "item_id": "gold_coin", "from": "all", "quantity": 0 } }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `item_id` | string | Yes | ID of the item to remove |
| `from` | string | Yes | `"current_player"`, `"all"`, or a specific player ID |
| `quantity` | number | No | Number to remove (default: `1`). Use `0` to remove all. |

### reveal_item (legacy)

Reveals an item to players. This is a legacy action kept for backward compatibility — it also adds the item to the player's inventory (quantity 1).

```json
{ "reveal_item": { "item_id": "old_diary", "to": "all" } }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `item_id` | string | Yes | ID of the item to reveal |
| `to` | string | Yes | `"current_player"`, `"all"`, or a specific player ID |

### reveal_npc_field

Reveals a hidden NPC field.

```json
{ "reveal_npc_field": { "npc_id": "ghost", "field_key": "secret", "to": "all" } }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `npc_id` | string | Yes | ID of the NPC |
| `field_key` | string | Yes | Key of the field to reveal |
| `to` | string | Yes | `"current_player"`, `"all"`, or a specific player ID |

---

## Rules

Optional custom rules for the scenario.

```json
{
  "attributes": [
    { "name": "courage", "display": "Courage", "default": 0 },
    { "name": "perception", "display": "Perception", "default": 0 }
  ],
  "dice_formula": "2d6+0",
  "check_method": "gte"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `attributes` | Attribute[] | No | Character attribute definitions |
| `dice_formula` | string | No | Default dice formula in `NdS+M` format |
| `check_method` | string | No | `"gte"` (≥ target) or `"gt"` (> target) |
| `gm_reference` | string | No | GM 快速參考（Markdown），顯示在 GM Console「規則」分頁 |

### Attribute

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Attribute key |
| `display` | string | Yes | Display name |
| `default` | number | Yes | Default value for new characters |

### Rule System Presets (規則預設)

The scenario editor provides two built-in rule system presets. Selecting a preset auto-fills attributes, dice formula, check method, GM reference, and suggested variables/items.

#### Wuxia — 武俠（江湖風雲錄）

Chinese martial arts fantasy. 4 attributes: **武功** (martial arts), **內力** (inner force), **身法** (agility), **機智** (wisdom).

Core mechanic: Roll `2d6 + attribute` ≥ target number (8/10/12/14).

Special mechanics:
- **Inner Force Burn**: Consume an "內力點" item for +2 on a check
- **Reputation**: Integer variable tracking jianghu standing
- **Secret Manuals**: Key items granting permanent +1 to an attribute
- **Duels**: Opposed `2d6 + martial` rolls, best of 3

#### Detective — 偵探推理（迷霧真相）

Investigation and deduction. 4 attributes: **觀察** (observe), **推理** (reason), **交際** (social), **膽識** (nerve).

Core mechanic: Roll `2d6 + attribute` ≥ target number (8/10/12/14).

Special mechanics:
- **Clue Quality**: Critical clues always found; dice determine bonus info depth
- **Pressure Points**: Consume a "壓力點" item to re-roll or +2 during interrogation
- **Clue Count**: Integer variable gating access to the deduction scene
- **Suspicion Tracking**: Per-suspect integer variables unlocking confrontation scenes

### Dice Formula Format

Format: `NdS+M` where:
- **N** = number of dice
- **S** = number of sides per die
- **M** = modifier (can be negative)

Examples:
- `2d6+0` — Roll 2 six-sided dice
- `1d20+3` — Roll 1 twenty-sided die, add 3
- `3d8-1` — Roll 3 eight-sided dice, subtract 1

---

## Complete Example

See the full working example at [`docs/sample-scenario.json`](./sample-scenario.json).

The sample scenario ("The Haunted Mansion") demonstrates all features:
- 6 scenes with branching paths
- 4 items with `gm_notes` (key_item, clue, consumable with `stackable`, treasure)
- 2 NPCs with public/hidden/gm_only fields
- 4 variables (bool and int)
- `on_enter` actions: `set_var`, `give_item`, `remove_item`, `reveal_npc_field`
- `player_choice` and `gm_decision` transitions
- Conditional transitions using `has_item()` function
- Inventory lifecycle: give key in kitchen → consume key entering secret room
- Multiple endings (good ending gives a reward item to all players)
- Dice rules (2d6, gte check)
