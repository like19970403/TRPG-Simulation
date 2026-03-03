import { useState, useEffect } from 'react'
import { BasicInfoSection } from './sections/basic-info-section'
import { ScenesSection } from './sections/scenes-section'
import { ItemsSection } from './sections/items-section'
import { NpcsSection } from './sections/npcs-section'
import { VariablesSection } from './sections/variables-section'
import { RulesSection } from './sections/rules-section'
import type { ScenarioContent } from '../../api/types'
import { cn } from '../../lib/cn'

type FormTab = 'basic' | 'scenes' | 'items' | 'npcs' | 'variables' | 'rules'

interface ContentFormEditorProps {
  data: ScenarioContent
  onChange: (data: ScenarioContent) => void
}

const tabs: { key: FormTab; label: string }[] = [
  { key: 'basic', label: '基本資訊' },
  { key: 'scenes', label: '場景' },
  { key: 'items', label: '道具' },
  { key: 'npcs', label: 'NPC' },
  { key: 'variables', label: '變數' },
  { key: 'rules', label: '規則' },
]

/** Normalize legacy trigger values (e.g. "gm" → "gm_decision") */
function normalizeData(d: ScenarioContent): ScenarioContent {
  let changed = false
  const scenes = d.scenes.map((scene) => {
    if (!scene.transitions) return scene
    const transitions = scene.transitions.map((t) => {
      if (t.trigger === 'gm') {
        changed = true
        return { ...t, trigger: 'gm_decision' }
      }
      return t
    })
    return { ...scene, transitions }
  })
  return changed ? { ...d, scenes } : d
}

export function ContentFormEditor({ data, onChange }: ContentFormEditorProps) {
  const [activeTab, setActiveTab] = useState<FormTab>('basic')

  // Normalize legacy data on first load
  useEffect(() => {
    const normalized = normalizeData(data)
    if (normalized !== data) onChange(normalized)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const allSceneIds = data.scenes.map((s) => s.id).filter(Boolean)
  const allVariableNames = data.variables.map((v) => v.name).filter(Boolean)
  const allVariables = data.variables.filter((v) => v.name)

  return (
    <div className="flex flex-col">
      {/* Tab bar */}
      <div className="flex border-b border-border">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            className={cn(
              'px-4 py-2 text-xs font-medium transition-colors',
              activeTab === tab.key
                ? 'border-b-2 border-gold text-gold'
                : 'text-text-tertiary hover:text-text-secondary',
            )}
            onClick={() => setActiveTab(tab.key)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="py-4">
        {activeTab === 'basic' && (
          <BasicInfoSection
            data={data}
            onChange={(patch) => onChange({ ...data, ...patch })}
          />
        )}

        {activeTab === 'scenes' && (
          <ScenesSection
            scenes={data.scenes}
            onChange={(scenes) => onChange({ ...data, scenes })}
            allSceneIds={allSceneIds}
            allItems={data.items}
            allNpcs={data.npcs}
            allVariableNames={allVariableNames}
            allVariables={allVariables}
          />
        )}

        {activeTab === 'items' && (
          <ItemsSection
            items={data.items}
            onChange={(items) => onChange({ ...data, items })}
          />
        )}

        {activeTab === 'npcs' && (
          <NpcsSection
            npcs={data.npcs}
            onChange={(npcs) => onChange({ ...data, npcs })}
          />
        )}

        {activeTab === 'variables' && (
          <VariablesSection
            variables={data.variables}
            onChange={(variables) => onChange({ ...data, variables })}
          />
        )}

        {activeTab === 'rules' && (
          <RulesSection
            rules={data.rules}
            onChange={(rules) => onChange({ ...data, rules })}
          />
        )}
      </div>
    </div>
  )
}
