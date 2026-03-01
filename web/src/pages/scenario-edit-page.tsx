import { type FormEvent, useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { FormField } from '../components/ui/form-field'
import { ContentEditor } from '../components/scenario/content-editor'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { ROUTES } from '../lib/constants'
import * as scenarioApi from '../api/scenarios'
import { ApiClientError } from '../api/client'
import type { ErrorDetail } from '../api/types'
import sampleScenario from '../../../docs/sample-scenario.json'

export function ScenarioEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEditMode = !!id

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [pageLoading, setPageLoading] = useState(isEditMode)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [generalError, setGeneralError] = useState('')

  useEffect(() => {
    if (!id) return
    scenarioApi
      .getScenario(id)
      .then((sc) => {
        setTitle(sc.title)
        setDescription(sc.description)
        setContent(JSON.stringify(sc.content, null, 2))
      })
      .catch((err) => {
        setGeneralError(
          err instanceof ApiClientError
            ? err.body.message
            : 'Failed to load scenario',
        )
      })
      .finally(() => setPageLoading(false))
  }, [id])

  const validate = (): boolean => {
    const errs: Record<string, string> = {}

    if (!title.trim()) {
      errs.title = 'Title is required'
    } else if (title.length > 200) {
      errs.title = 'Title must be 200 characters or less'
    }

    if (content.trim()) {
      try {
        const parsed = JSON.parse(content)
        if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
          errs.content = 'Content must be a JSON object'
        }
      } catch {
        errs.content = 'Invalid JSON syntax'
      }
    }

    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setGeneralError('')
    if (!validate()) return

    setLoading(true)
    try {
      const parsedContent = content.trim() ? JSON.parse(content) : {}
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
        setGeneralError('An unexpected error occurred')
      }
    } finally {
      setLoading(false)
    }
  }

  function handleLoadSample() {
    const hasContent = title.trim() || description.trim() || content.trim()
    if (hasContent && !confirm('This will replace your current content. Continue?')) {
      return
    }
    setTitle(sampleScenario.title)
    setDescription('A haunted mansion adventure featuring multiple scenes, items, NPCs, dice checks, and multiple endings.')
    setContent(JSON.stringify(sampleScenario, null, 2))
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
          &larr; Cancel
        </Link>
        <h1 className="font-display text-2xl font-semibold text-text-primary">
          {isEditMode ? 'Edit Scenario' : 'New Scenario'}
        </h1>
        <div className="flex gap-3">
          {!isEditMode && (
            <Button variant="secondary" onClick={handleLoadSample}>
              Load Sample
            </Button>
          )}
          <Button onClick={handleSubmit} loading={loading}>
            Save Draft
          </Button>
        </div>
      </div>

      {/* Errors */}
      {generalError && (
        <p className="rounded-lg bg-error/10 px-4 py-3 text-sm text-error">
          {generalError}
        </p>
      )}

      {/* Form */}
      <form onSubmit={handleSubmit} className="flex flex-col gap-6">
        <FormField label="Title" error={errors.title}>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Enter scenario title"
            error={!!errors.title}
            maxLength={200}
          />
        </FormField>

        <FormField label="Description" error={errors.description}>
          <Input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Brief description of the scenario"
          />
        </FormField>

        <FormField label="Content (JSON)">
          <ContentEditor
            value={content}
            onChange={setContent}
            error={errors.content}
          />
        </FormField>
      </form>
    </div>
  )
}
