import { useState, useEffect, useCallback } from 'react'
import { Button } from '../components/ui/button'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { SessionCard } from '../components/session/session-card'
import { JoinSessionModal } from '../components/session/join-session-modal'
import * as sessionApi from '../api/sessions'
import * as scenarioApi from '../api/scenarios'
import type { SessionResponse } from '../api/types'
import { ApiClientError } from '../api/client'

export function SessionListPage() {
  const [sessions, setSessions] = useState<SessionResponse[]>([])
  const [scenarioTitles, setScenarioTitles] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showJoinModal, setShowJoinModal] = useState(false)

  const fetchSessions = useCallback(async () => {
    setLoading(true)
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
            titles[id] = 'Unknown Scenario'
          }
        }),
      )
      setScenarioTitles(titles)
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.body.message)
      } else {
        setError('Failed to load sessions')
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchSessions()
  }, [fetchSessions])

  return (
    <div className="flex flex-col gap-8 px-[60px] py-10">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="font-display text-[32px] font-semibold text-text-primary">
          Sessions
        </h1>
        <Button onClick={() => setShowJoinModal(true)}>Join Session</Button>
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex justify-center py-12">
          <LoadingSpinner className="h-8 w-8 text-gold" />
        </div>
      ) : error ? (
        <p className="py-8 text-center text-sm text-error">{error}</p>
      ) : sessions.length === 0 ? (
        <p className="py-8 text-center text-sm text-text-tertiary">
          No sessions yet
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {sessions.map((session) => (
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
