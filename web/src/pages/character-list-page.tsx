import { useState, useEffect, useCallback } from 'react'
import { Button } from '../components/ui/button'
import { LoadingSpinner } from '../components/ui/loading-spinner'
import { CharacterCard } from '../components/character/character-card'
import { CharacterFormModal } from '../components/character/character-form-modal'
import { listCharacters, deleteCharacter } from '../api/characters'
import { ApiClientError } from '../api/client'
import type { CharacterResponse } from '../api/types'

export function CharacterListPage() {
  const [characters, setCharacters] = useState<CharacterResponse[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingChar, setEditingChar] = useState<CharacterResponse | null>(null)

  const fetchCharacters = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listCharacters(50, 0)
      setCharacters(res.characters)
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.body.message)
      } else {
        setError('角色載入失敗')
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchCharacters()
  }, [fetchCharacters])

  function handleEdit(character: CharacterResponse) {
    setEditingChar(character)
    setShowModal(true)
  }

  async function handleDelete(character: CharacterResponse) {
    if (!confirm(`確定要刪除「${character.name}」？`)) return
    try {
      await deleteCharacter(character.id)
      await fetchCharacters()
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.body.message)
      } else {
        setError('刪除角色失敗')
      }
    }
  }

  function handleCloseModal() {
    setShowModal(false)
    setEditingChar(null)
  }

  function handleSaved() {
    fetchCharacters()
  }

  return (
    <div className="flex flex-col gap-8 px-[60px] py-10">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="font-display text-[32px] font-semibold text-text-primary">
          角色
        </h1>
        <Button onClick={() => setShowModal(true)}>+ 新增角色</Button>
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex justify-center py-12">
          <LoadingSpinner className="h-8 w-8 text-gold" />
        </div>
      ) : error ? (
        <p className="py-8 text-center text-sm text-error">{error}</p>
      ) : characters.length === 0 ? (
        <p className="py-8 text-center text-sm text-text-tertiary">
          還沒有角色，建立你的第一個吧！
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {characters.map((char) => (
            <CharacterCard
              key={char.id}
              character={char}
              onEdit={handleEdit}
              onDelete={handleDelete}
            />
          ))}
        </div>
      )}

      {/* Create/Edit Modal */}
      <CharacterFormModal
        open={showModal}
        onClose={handleCloseModal}
        onSaved={handleSaved}
        character={editingChar}
      />
    </div>
  )
}
