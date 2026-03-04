import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { ScenarioStatusBadge } from '../components/scenario/scenario-status-badge'
import { ScenarioToolbar } from '../components/scenario/scenario-toolbar'
import { ConfirmModal } from '../components/scenario/confirm-modal'
import { ROUTES } from '../lib/constants'
import * as scenarioApi from '../api/scenarios'
import { createSession } from '../api/sessions'
import type { ScenarioResponse } from '../api/types'
import { ApiClientError } from '../api/client'

function countContentStats(content: Record<string, unknown>) {
  const scenes = Array.isArray(content.scenes) ? content.scenes.length : 0
  const items = Array.isArray(content.items) ? content.items.length : 0
  const npcs = Array.isArray(content.npcs) ? content.npcs.length : 0
  return { scenes, items, npcs }
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('zh-TW', {
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
    'publish' | 'unpublish' | 'archive' | 'delete' | null
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
            : '劇本載入失敗',
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
      } else if (modalType === 'unpublish') {
        const updated = await scenarioApi.unpublishScenario(id)
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
        err instanceof ApiClientError ? err.body.message : '操作失敗',
      )
      setModalType(null)
    } finally {
      setActionLoading(false)
    }
  }

  const handleHostGame = async () => {
    if (!id) return
    setActionLoading(true)
    try {
      const session = await createSession({ scenarioId: id })
      navigate(`/sessions/${session.id}/lobby`)
    } catch (err) {
      setError(
        err instanceof ApiClientError ? err.body.message : '建立場次失敗',
      )
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
        <p className="text-sm text-error">{error || '找不到劇本'}</p>
        <Link to={ROUTES.SCENARIOS} className="text-sm text-gold hover:text-gold-light">
          回到劇本列表
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
        &larr; 回到劇本列表
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
          onUnpublish={() => setModalType('unpublish')}
          onArchive={() => setModalType('archive')}
          onDelete={() => setModalType('delete')}
          onHostGame={handleHostGame}
        />
      </div>

      {/* Metadata */}
      <div className="flex items-center gap-4">
        <ScenarioStatusBadge status={scenario.status} />
        <span className="text-sm text-text-tertiary">
          版本 {scenario.version}
        </span>
        <span className="text-sm text-text-tertiary">
          更新於 {formatDate(scenario.updatedAt)}
        </span>
      </div>

      {/* Description */}
      {scenario.description && (
        <div className="flex flex-col gap-2">
          <h2 className="text-sm font-semibold text-text-secondary">
            描述
          </h2>
          <p className="text-sm text-text-primary">{scenario.description}</p>
        </div>
      )}

      {/* Content preview */}
      <div className="flex flex-col gap-2">
        <h2 className="text-sm font-semibold text-text-secondary">
          內容預覽
        </h2>
        <pre className="max-h-[400px] overflow-auto rounded-md border border-border bg-[#1A1A1A] p-4 font-mono text-xs text-text-primary">
          {JSON.stringify(scenario.content, null, 2)}
        </pre>
      </div>

      {/* Stats */}
      <div className="flex items-center gap-6 text-sm text-text-secondary">
        <span>場景：{stats.scenes}</span>
        <span>道具：{stats.items}</span>
        <span>NPC：{stats.npcs}</span>
      </div>

      {/* Confirm modals */}
      <ConfirmModal
        open={modalType === 'publish'}
        onClose={() => setModalType(null)}
        onConfirm={handleConfirm}
        title="發布劇本？"
        description="發布後可供遊戲場次使用。如需修改，可隨時取消發布回到草稿狀態。"
        confirmLabel="發布"
        loading={actionLoading}
      />
      <ConfirmModal
        open={modalType === 'unpublish'}
        onClose={() => setModalType(null)}
        onConfirm={handleConfirm}
        title="取消發布？"
        description="劇本將回到草稿狀態，可以重新編輯。已使用此劇本的場次不受影響。"
        confirmLabel="取消發布"
        confirmVariant="secondary"
        loading={actionLoading}
      />
      <ConfirmModal
        open={modalType === 'archive'}
        onClose={() => setModalType(null)}
        onConfirm={handleConfirm}
        title="封存劇本？"
        description="封存後的劇本為唯讀，並從活動列表中隱藏。已使用此劇本的場次不受影響。"
        confirmLabel="封存"
        confirmVariant="secondary"
        loading={actionLoading}
      />
      <ConfirmModal
        open={modalType === 'delete'}
        onClose={() => setModalType(null)}
        onConfirm={handleConfirm}
        title="刪除劇本？"
        description="此操作無法復原。劇本及其所有內容將被永久移除。"
        confirmLabel="刪除"
        confirmClassName="bg-error hover:bg-error/80 text-white"
        loading={actionLoading}
      />
    </div>
  )
}
