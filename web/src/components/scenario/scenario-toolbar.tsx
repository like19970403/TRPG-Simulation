import type { ScenarioStatus } from '../../api/types'
import { Button } from '../ui/button'

interface ScenarioToolbarProps {
  status: ScenarioStatus
  onEdit: () => void
  onPublish: () => void
  onArchive: () => void
  onDelete: () => void
}

export function ScenarioToolbar({
  status,
  onEdit,
  onPublish,
  onArchive,
  onDelete,
}: ScenarioToolbarProps) {
  return (
    <div className="flex items-center gap-2.5">
      {status === 'draft' && (
        <>
          <Button variant="ghost" size="sm" onClick={onEdit}>
            Edit
          </Button>
          <Button variant="primary" size="sm" onClick={onPublish}>
            Publish
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="text-error hover:text-error"
            onClick={onDelete}
          >
            Delete
          </Button>
        </>
      )}
      {status === 'published' && (
        <Button variant="secondary" size="sm" onClick={onArchive}>
          Archive
        </Button>
      )}
    </div>
  )
}
