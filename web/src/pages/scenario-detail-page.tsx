import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { ScenarioStatusBadge } from '../components/scenario/scenario-status-badge'
import { ScenarioToolbar } from '../components/scenario/scenario-toolbar'
import { ConfirmModal } from '../components/scenario/confirm-modal'
import { ROUTES } from '../lib/constants'
import * as scenarioApi from '../api/scenarios'
import type { ScenarioResponse } from '../api/types'
import { ApiClientError } from '../api/client'

function countContentStats(content: Record<string, unknown>) {
  const scenes = Array.isArray(content.scenes) ? content.scenes.length : 0
  const items = Array.isArray(content.items) ? content.items.length : 0
  const npcs = Array.isArray(content.npcs) ? content.npcs.length : 0
  return { scenes, items, npcs }
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function ScenarioDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [scenario, setScenario] = useState<ScenarioResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [modalType, setModalType] = useState<
    'publish' | 'archive' | 'delete' | null
  >(null)
  const [actionLoading, setActionLoading] = useState(false)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    scenarioApi
      .getScenario(id)
      .then(setScenario)
      .catch((err) => {
        setError(
          err instanceof ApiClientError
            ? err.body.message
            : 'Failed to load scenario',
        )
      })
      .finally(() => setLoading(false))
  }, [id])

  const stats = useMemo(
    () =>
      scenario
        ? countContentStats(scenario.content)
        : { scenes: 0, items: 0, npcs: 0 },
    [scenario],
  )

  const handleConfirm = async () => {
    if (!id || !modalType) return
    setActionLoading(true)
    try {
      if (modalType === 'publish') {
        const updated = await scenarioApi.publishScenario(id)
        setScenario(updated)
      } else if (modalType === 'archive') {
        const updated = await scenarioApi.archiveScenario(id)
        setScenario(updated)
      } else if (modalType === 'delete') {
        await scenarioApi.deleteScenario(id)
        navigate(ROUTES.SCENARIOS)
        return
      }
      setModalType(null)
    } catch (err) {
      setError(
        err instanceof ApiClientError ? err.body.message : 'Action failed',
      )
      setModalType(null)
    } finally {
      setActionLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-24">
        <LoadingSpinner className="h-8 w-8 text-gold" />
      </div>
    )
  }

  if (error || !scenario) {
    return (
      <div className="flex flex-col items-center gap-4 py-24">
        <p className="text-sm text-error">{error || 'Scenario not found'}</p>
        <Link to={ROUTES.SCENARIOS} className="text-sm text-gold hover:text-gold-light">
          Back to Scenarios
        </Link>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-8 px-[60px] py-10">
      {/* Back link */}
      <Link
        to={ROUTES.SCENARIOS}
        className="text-sm text-text-secondary hover:text-text-primary"
      >
        &larr; Back to Scenarios
      </Link>

      {/* Title row */}
      <div className="flex items-center justify-between">
        <h1 className="font-display text-[28px] font-semibold text-text-primary">
          {scenario.title}
        </h1>
        <ScenarioToolbar
          status={scenario.status}
          onEdit={() => navigate(`/scenarios/${id}/edit`)}
          onPublish={() => setModalType('publish')}
          onArchive={() => setModalType('archive')}
          onDelete={() => setModalType('delete')}
        />
      </div>

      {/* Metadata */}
      <div className="flex items-center gap-4">
        <ScenarioStatusBadge status={scenario.status} />
        <span className="text-sm text-text-tertiary">
          Version {scenario.version}
        </span>
        <span className="text-sm text-text-tertiary">
          Updated {formatDate(scenario.updatedAt)}
        </span>
      </div>

      {/* Description */}
      {scenario.description && (
        <div className="flex flex-col gap-2">
          <h2 className="text-sm font-semibold text-text-secondary">
            Description
          </h2>
          <p className="text-sm text-text-primary">{scenario.description}</p>
        </div>
      )}

      {/* Content preview */}
      <div className="flex flex-col gap-2">
        <h2 className="text-sm font-semibold text-text-secondary">
          Content Preview
        </h2>
        <pre className="max-h-[400px] overflow-auto rounded-md border border-border bg-[#1A1A1A] p-4 font-mono text-xs text-text-primary">
          {JSON.stringify(scenario.content, null, 2)}
        </pre>
      </div>

      {/* Stats */}
      <div className="flex items-center gap-6 text-sm text-text-secondary">
        <span>Scenes: {stats.scenes}</span>
        <span>Items: {stats.items}</span>
        <span>NPCs: {stats.npcs}</span>
      </div>

      {/* Confirm modals */}
      <ConfirmModal
        open={modalType === 'publish'}
        onClose={() => setModalType(null)}
        onConfirm={handleConfirm}
        title="Publish Scenario?"
        description="Once published, the scenario cannot be edited. Players will be able to use it in game sessions."
        confirmLabel="Publish"
        loading={actionLoading}
      />
      <ConfirmModal
        open={modalType === 'archive'}
        onClose={() => setModalType(null)}
        onConfirm={handleConfirm}
        title="Archive Scenario?"
        description="Archived scenarios are read-only and hidden from the active list. Existing sessions using this scenario will not be affected."
        confirmLabel="Archive"
        confirmVariant="secondary"
        loading={actionLoading}
      />
      <ConfirmModal
        open={modalType === 'delete'}
        onClose={() => setModalType(null)}
        onConfirm={handleConfirm}
        title="Delete Scenario?"
        description="This action cannot be undone. The scenario and all its content will be permanently removed."
        confirmLabel="Delete"
        confirmClassName="bg-error hover:bg-error/80 text-white"
        loading={actionLoading}
      />
    </div>
  )
}
