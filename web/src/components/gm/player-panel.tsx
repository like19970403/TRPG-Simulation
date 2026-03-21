import { useState } from 'react'
import { useGameStore } from '../../stores/game-store'
import { cn } from '../../lib/cn'
import { HelpIcon } from '../ui/tooltip'
import { CharacterCardModal } from '../player/character-card-modal'

const EMPTY_ATTRS: Record<string, Record<string, unknown>> = {}

export function PlayerPanel() {
  const players = useGameStore((s) => s.gameState?.players)
  const playerAttributes = useGameStore(
    (s) => s.gameState?.player_attributes ?? EMPTY_ATTRS,
  )
  const rules = useGameStore((s) => s.scenarioContent?.rules)
  const [selectedPlayer, setSelectedPlayer] = useState<string | null>(null)

  const playerList = players ? Object.entries(players) : []
  const onlineCount = playerList.filter(([, p]) => p.online).length

  // Build attribute display name map from rules
  const attrDisplayMap: Record<string, string> = {}
  if (rules?.attributes) {
    for (const attr of rules.attributes) {
      attrDisplayMap[attr.name] = attr.display
    }
  }

  return (
    <div className="flex h-full w-full flex-col overflow-y-auto bg-bg-sidebar p-5 lg:w-65">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="flex items-center gap-1.5 font-display text-sm font-semibold uppercase tracking-wider text-gold">
          玩家
          <HelpIcon tip="顯示已連線玩家及其角色屬性。灰色表示離線。" />
        </h2>
        <span className="rounded-full bg-gold-tint px-2 py-0.5 text-xs font-medium text-gold">
          {onlineCount}
        </span>
      </div>

      {playerList.length === 0 ? (
        <p className="text-xs text-text-tertiary">尚未有玩家連線</p>
      ) : (
        <ul className="flex flex-col gap-1">
          {playerList.map(([playerId, player]) => {
            const attrs = playerAttributes[playerId]
            const attrEntries = attrs ? Object.entries(attrs) : []

            return (
              <li
                key={playerId}
                className="rounded-lg px-3 py-2 hover:bg-bg-input"
              >
                {/* Player name row */}
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        'h-2 w-2 shrink-0 rounded-full',
                        player.online ? 'bg-success' : 'bg-text-tertiary',
                      )}
                    />
                    <div className="flex flex-col">
                      <span
                        className={cn(
                          'text-sm font-medium',
                          player.online
                            ? 'text-text-primary'
                            : 'text-text-tertiary',
                        )}
                      >
                        {player.character_name || player.username}
                        {!player.online && (
                          <span className="ml-1 text-xs font-normal text-text-tertiary">
                            （離線）
                          </span>
                        )}
                      </span>
                      {player.character_name && (
                        <span className="text-xs text-text-tertiary">
                          {player.username}
                        </span>
                      )}
                    </div>
                  </div>
                  {player.character_name && (
                    <button
                      type="button"
                      onClick={() => setSelectedPlayer(playerId)}
                      className="rounded border border-border px-1.5 py-0.5 text-[9px] text-text-tertiary transition-colors hover:border-gold hover:text-gold"
                    >
                      角色卡
                    </button>
                  )}
                </div>

                {/* Attributes */}
                {attrEntries.length > 0 && (
                  <div className="ml-4 mt-1 flex flex-wrap gap-x-3 gap-y-0.5">
                    {attrEntries.map(([key, val]) => (
                      <span key={key} className="text-xs text-text-secondary">
                        <span className="text-text-tertiary">
                          {attrDisplayMap[key] ?? key}
                        </span>{' '}
                        {String(val)}
                      </span>
                    ))}
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}

      {/* Character card modal */}
      {selectedPlayer && (
        <CharacterCardModal
          userId={selectedPlayer}
          onClose={() => setSelectedPlayer(null)}
        />
      )}
    </div>
  )
}
