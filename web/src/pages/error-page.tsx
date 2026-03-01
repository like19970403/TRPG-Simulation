import { useRouteError, useNavigate } from 'react-router'

export function ErrorPage() {
  const error = useRouteError()
  const navigate = useNavigate()

  console.error('[ErrorPage]', error)

  return (
    <div className="flex h-screen flex-col items-center justify-center gap-4 bg-bg-page">
      <div className="text-center">
        <h1 className="mb-2 text-lg font-semibold text-text-primary">
          發生錯誤
        </h1>
        <p className="mb-6 text-sm text-text-tertiary">
          發生未預期的錯誤，請重試。
        </p>
        <div className="flex gap-3 justify-center">
          <button
            className="rounded border border-border px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-sidebar"
            onClick={() => navigate(0)}
          >
            重試
          </button>
          <button
            className="rounded bg-gold px-4 py-2 text-sm font-medium text-bg-page transition-opacity hover:opacity-90"
            onClick={() => navigate('/')}
          >
            回到儀表板
          </button>
        </div>
      </div>
    </div>
  )
}
