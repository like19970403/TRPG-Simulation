import { Outlet } from 'react-router'

export function AuthLayout() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-bg-page px-4">
      <div className="w-full max-w-[420px] rounded-xl border border-border bg-bg-sidebar p-10">
        <div className="mb-6 flex items-center justify-center gap-2">
          <span className="text-2xl">📜</span>
          <h1 className="font-display text-[28px] font-semibold text-gold">
            TRPG
          </h1>
        </div>
        <Outlet />
      </div>
    </div>
  )
}
