import { useState, useEffect, useCallback } from 'react'
import { Link, useSearchParams } from 'react-router'
import { Button } from '../components/ui/button'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { ScenarioCard } from '../components/scenario/scenario-card'
import { ROUTES } from '../lib/constants'
import { cn } from '../lib/cn'
import * as scenarioApi from '../api/scenarios'
import type { ScenarioResponse, ScenarioStatus } from '../api/types'
import { ApiClientError } from '../api/client'

type TabFilter = 'all' | ScenarioStatus

const TABS: { label: string; value: TabFilter }[] = [
  { label: '全部', value: 'all' },
  { label: '草稿', value: 'draft' },
  { label: '已發布', value: 'published' },
  { label: '已封存', value: 'archived' },
]

const PAGE_SIZE = 20

export function ScenarioListPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [scenarios, setScenarios] = useState<ScenarioResponse[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [activeTab, setActiveTab] = useState<TabFilter>('all')

  const offset = parseInt(searchParams.get('offset') ?? '0', 10)

  const fetchScenarios = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await scenarioApi.listScenarios(PAGE_SIZE, offset)
      setScenarios(res.scenarios)
      setTotal(res.total)
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.body.message)
      } else {
        setError('劇本載入失敗')
      }
    } finally {
      setLoading(false)
    }
  }, [offset])

  useEffect(() => {
    fetchScenarios()
  }, [fetchScenarios])

  const filteredScenarios =
    activeTab === 'all'
      ? scenarios
      : scenarios.filter((s) => s.status === activeTab)

  const handlePrev = () => {
    const newOffset = Math.max(0, offset - PAGE_SIZE)
    setSearchParams({ offset: String(newOffset) })
  }

  const handleNext = () => {
    setSearchParams({ offset: String(offset + PAGE_SIZE) })
  }

  return (
    <div className="flex flex-col gap-8 px-[60px] py-10">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="font-display text-[32px] font-semibold text-text-primary">
          劇本
        </h1>
        <Link to={ROUTES.SCENARIO_NEW}>
          <Button>+ 新增劇本</Button>
        </Link>
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
      ) : filteredScenarios.length === 0 ? (
        <p className="py-8 text-center text-sm text-text-tertiary">
          {activeTab === 'all'
            ? '還沒有劇本，建立你的第一個吧！'
            : '沒有符合此篩選的劇本。'}
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {filteredScenarios.map((scenario) => (
            <ScenarioCard key={scenario.id} scenario={scenario} />
          ))}
        </div>
      )}

      {/* Pagination */}
      {!loading && total > 0 && (
        <div className="flex items-center justify-between">
          <span className="text-[13px] text-text-tertiary">
            顯示 {offset + 1}-{Math.min(offset + PAGE_SIZE, total)} / 共{' '}
            {total} 個
          </span>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={handlePrev}
              disabled={offset === 0}
            >
              上一頁
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleNext}
              disabled={offset + PAGE_SIZE >= total}
            >
              下一頁
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
