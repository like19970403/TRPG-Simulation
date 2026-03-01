import { type FormEvent, useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { FormField } from '../components/ui/form-field'
import { ContentEditor } from '../components/scenario/content-editor'
import { ContentFormEditor } from '../components/scenario/content-form-editor'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { ROUTES } from '../lib/constants'
import * as scenarioApi from '../api/scenarios'
import { ApiClientError } from '../api/client'
import type { ErrorDetail, ScenarioContent } from '../api/types'
import sampleScenario from '../../../docs/sample-scenario.json'
import { cn } from '../lib/cn'

const defaultFormData: ScenarioContent = {
  id: '',
  title: '',
  start_scene: '',
  scenes: [],
  items: [],
  npcs: [],
  variables: [],
}

function tryParseContent(json: string): ScenarioContent | null {
  try {
    const parsed = JSON.parse(json)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      return null
    }
    return {
      id: parsed.id ?? '',
      title: parsed.title ?? '',
      start_scene: parsed.start_scene ?? '',
      scenes: Array.isArray(parsed.scenes) ? parsed.scenes : [],
      items: Array.isArray(parsed.items) ? parsed.items : [],
      npcs: Array.isArray(parsed.npcs) ? parsed.npcs : [],
      variables: Array.isArray(parsed.variables) ? parsed.variables : [],
      rules: parsed.rules ?? undefined,
    }
  } catch {
    return null
  }
}

export function ScenarioEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEditMode = !!id

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [content, setContent] = useState('')
  const [formData, setFormData] = useState<ScenarioContent>(defaultFormData)
  const [editorMode, setEditorMode] = useState<'form' | 'json'>('form')
  const [loading, setLoading] = useState(false)
  const [pageLoading, setPageLoading] = useState(isEditMode)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [generalError, setGeneralError] = useState('')
  const [switchError, setSwitchError] = useState('')

  useEffect(() => {
    if (!id) return
    scenarioApi
      .getScenario(id)
      .then((sc) => {
        setTitle(sc.title)
        setDescription(sc.description)
        const jsonStr = JSON.stringify(sc.content, null, 2)
        setContent(jsonStr)
        const parsed = tryParseContent(jsonStr)
        if (parsed) {
          setFormData(parsed)
          setEditorMode('form')
        } else {
          setEditorMode('json')
        }
      })
      .catch((err) => {
        setGeneralError(
          err instanceof ApiClientError
            ? err.body.message
            : '劇本載入失敗',
        )
      })
      .finally(() => setPageLoading(false))
  }, [id])

  const switchToJson = () => {
    setSwitchError('')
    const jsonStr = JSON.stringify(formData, null, 2)
    setContent(jsonStr)
    setEditorMode('json')
  }

  const switchToForm = () => {
    setSwitchError('')
    const parsed = tryParseContent(content)
    if (parsed) {
      setFormData(parsed)
      setEditorMode('form')
    } else {
      setSwitchError('JSON 格式不正確，無法切換至表單模式')
    }
  }

  const validate = (): boolean => {
    const errs: Record<string, string> = {}

    if (!title.trim()) {
      errs.title = '標題為必填'
    } else if (title.length > 200) {
      errs.title = '標題不可超過 200 個字元'
    }

    if (editorMode === 'json' && content.trim()) {
      try {
        const parsed = JSON.parse(content)
        if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
          errs.content = '內容必須是 JSON 物件'
        }
      } catch {
        errs.content = 'JSON 語法錯誤'
      }
    }

    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  const handleSubmit = async (e?: FormEvent) => {
    e?.preventDefault()
    setGeneralError('')
    if (!validate()) return

    setLoading(true)
    try {
      let parsedContent: Record<string, unknown>
      if (editorMode === 'form') {
        parsedContent = formData as unknown as Record<string, unknown>
      } else {
        parsedContent = content.trim() ? JSON.parse(content) : {}
      }

      const data = {
        title: title.trim(),
        description: description.trim(),
        content: parsedContent,
      }

      if (isEditMode && id) {
        await scenarioApi.updateScenario(id, data)
        navigate(`/scenarios/${id}`)
      } else {
        const created = await scenarioApi.createScenario(data)
        navigate(`/scenarios/${created.id}`)
      }
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.body.details?.length) {
          const fieldErrors: Record<string, string> = {}
          err.body.details.forEach((d: ErrorDetail) => {
            fieldErrors[d.field] = d.reason
          })
          setErrors(fieldErrors)
        } else {
          setGeneralError(err.body.message)
        }
      } else {
        setGeneralError('發生未預期的錯誤')
      }
    } finally {
      setLoading(false)
    }
  }

  function handleLoadSample() {
    const hasContent =
      title.trim() || description.trim() || content.trim() ||
      formData.scenes.length > 0
    if (hasContent && !confirm('這將會取代你目前的內容，確定要繼續嗎？')) {
      return
    }
    setTitle(sampleScenario.title)
    setDescription('一場鬧鬼大宅冒險，包含多個場景、道具、NPC、骰子檢定與多重結局。')
    const jsonStr = JSON.stringify(sampleScenario, null, 2)
    setContent(jsonStr)
    const parsed = tryParseContent(jsonStr)
    if (parsed) {
      setFormData(parsed)
    }
  }

  if (pageLoading) {
    return (
      <div className="flex justify-center py-24">
        <LoadingSpinner className="h-8 w-8 text-gold" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-8 px-15 py-10">
      {/* Header */}
      <div className="flex items-center justify-between">
        <Link
          to={isEditMode ? `/scenarios/${id}` : ROUTES.SCENARIOS}
          className="text-sm text-text-secondary hover:text-text-primary"
        >
          &larr; 取消
        </Link>
        <h1 className="font-display text-2xl font-semibold text-text-primary">
          {isEditMode ? '編輯劇本' : '新增劇本'}
        </h1>
        <div className="flex gap-3">
          {!isEditMode && (
            <Button variant="secondary" onClick={handleLoadSample}>
              載入範例
            </Button>
          )}
          <Button onClick={() => handleSubmit()} loading={loading}>
            儲存草稿
          </Button>
        </div>
      </div>

      {/* Errors */}
      {generalError && (
        <p className="rounded-lg bg-error/10 px-4 py-3 text-sm text-error">
          {generalError}
        </p>
      )}

      {/* Form: title + description */}
      <form onSubmit={handleSubmit} className="flex flex-col gap-6">
        <FormField label="標題" error={errors.title}>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="輸入劇本標題"
            error={!!errors.title}
            maxLength={200}
          />
        </FormField>

        <FormField label="描述" error={errors.description}>
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="簡短描述劇本內容"
          />
        </FormField>

        {/* Mode toggle */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-4">
            <span className="text-xs font-medium text-text-secondary">
              內容編輯
            </span>
            <div className="flex border-b border-border">
              <button
                type="button"
                className={cn(
                  'px-4 py-2 text-xs font-medium transition-colors',
                  editorMode === 'form'
                    ? 'border-b-2 border-gold text-gold'
                    : 'text-text-tertiary hover:text-text-secondary',
                )}
                onClick={() =>
                  editorMode === 'json' ? switchToForm() : undefined
                }
              >
                表單模式
              </button>
              <button
                type="button"
                className={cn(
                  'px-4 py-2 text-xs font-medium transition-colors',
                  editorMode === 'json'
                    ? 'border-b-2 border-gold text-gold'
                    : 'text-text-tertiary hover:text-text-secondary',
                )}
                onClick={() =>
                  editorMode === 'form' ? switchToJson() : undefined
                }
              >
                JSON 模式
              </button>
            </div>
          </div>

          {switchError && (
            <p className="text-xs text-error">{switchError}</p>
          )}

          {/* Editor content */}
          {editorMode === 'form' ? (
            <ContentFormEditor data={formData} onChange={setFormData} />
          ) : (
            <ContentEditor
              value={content}
              onChange={setContent}
              error={errors.content}
            />
          )}
        </div>
      </form>
    </div>
  )
}
