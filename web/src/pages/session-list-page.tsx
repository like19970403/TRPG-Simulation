import { useState, useEffect, useCallback, useRef } from 'react'
import { Button } from '../components/ui/button'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { SessionCard } from '../components/session/session-card'
import { JoinSessionModal } from '../components/session/join-session-modal'
import { cn } from '../lib/cn'
import * as sessionApi from '../api/sessions'
import * as scenarioApi from '../api/scenarios'
import type { SessionResponse, SessionStatus } from '../api/types'
import { ApiClientError } from '../api/client'

type TabFilter = 'all' | SessionStatus

const TABS: { label: string; value: TabFilter }[] = [
  { label: '全部', value: 'all' },
  { label: '等待中', value: 'lobby' },
  { label: '進行中', value: 'active' },
  { label: '已結束', value: 'completed' },
]

export function SessionListPage() {
  const [sessions, setSessions] = useState<SessionResponse[]>([])
  const [scenarioTitles, setScenarioTitles] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showJoinModal, setShowJoinModal] = useState(false)
  const [activeTab, setActiveTab] = useState<TabFilter>('all')
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const hasFetched = useRef(false)

  const fetchSessions = useCallback(async () => {
    if (!hasFetched.current) setLoading(true)
    setError('')
    try {
      const res = await sessionApi.listSessions(50, 0)
      setSessions(res.sessions)

      // Fetch scenario titles for each unique scenarioId
      const uniqueIds = [...new Set(res.sessions.map((s) => s.scenarioId))]
      const titles: Record<string, string> = {}
      await Promise.allSettled(
        uniqueIds.map(async (id) => {
          try {
            const scenario = await scenarioApi.getScenario(id)
            titles[id] = scenario.title
          } catch {
            titles[id] = '未知劇本'
          }
        }),
      )
      setScenarioTitles(titles)
      setLastUpdated(new Date())
      hasFetched.current = true
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.body.message)
      } else {
        setError('場次載入失敗')
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchSessions()
    intervalRef.current = setInterval(fetchSessions, 30_000)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [fetchSessions])

  const filteredSessions =
    activeTab === 'all'
      ? sessions
      : sessions.filter((s) => s.status === activeTab)

  return (
    <div className="flex flex-col gap-6 px-4 py-6 md:gap-8 md:px-15 md:py-10">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="font-display text-2xl font-semibold text-text-primary md:text-[32px]">
            場次
          </h1>
          <button
            onClick={fetchSessions}
            disabled={loading}
            className="rounded-md px-2 py-1 text-xs text-text-tertiary transition-colors hover:bg-bg-surface hover:text-text-secondary disabled:opacity-50"
          >
            {loading ? '更新中...' : '重新整理'}
          </button>
          {lastUpdated && (
            <span className="text-xs text-text-tertiary">
              最後更新 {lastUpdated.toLocaleTimeString()}
            </span>
          )}
        </div>
        <Button onClick={() => setShowJoinModal(true)}>加入場次</Button>
      </div>

      {/* Tabs */}
      <div>
        <div className="flex">
          {TABS.map((tab) => (
            <button
              key={tab.value}
              onClick={() => setActiveTab(tab.value)}
              className={cn(
                'cursor-pointer px-4 py-2 text-sm font-medium transition-colors',
                activeTab === tab.value
                  ? 'rounded-t-md bg-gold-tint-30 text-text-primary'
                  : 'text-text-tertiary hover:text-text-secondary',
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="h-px w-full bg-border" />
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex justify-center py-12">
          <LoadingSpinner className="h-8 w-8 text-gold" />
        </div>
      ) : error ? (
        <p className="py-8 text-center text-sm text-error">{error}</p>
      ) : filteredSessions.length === 0 ? (
        <p className="py-8 text-center text-sm text-text-tertiary">
          {activeTab === 'all'
            ? '還沒有場次'
            : '沒有符合此篩選的場次'}
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {filteredSessions.map((session) => (
            <SessionCard
              key={session.id}
              session={session}
              scenarioTitle={scenarioTitles[session.scenarioId]}
            />
          ))}
        </div>
      )}

      {/* Join Session Modal */}
      <JoinSessionModal
        open={showJoinModal}
        onClose={() => setShowJoinModal(false)}
        onJoined={fetchSessions}
      />
    </div>
  )
}
