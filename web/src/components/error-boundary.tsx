import { Component } from 'react'
import type { ReactNode, ErrorInfo } from 'react'
import { useNavigate } from 'react-router'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

class ErrorBoundaryInner extends Component<
  Props & { onReset: () => void },
  State
> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-screen flex-col items-center justify-center gap-4 bg-bg-page">
          <div className="text-center">
            <h1 className="mb-2 text-lg font-semibold text-text-primary">
              Something went wrong
            </h1>
            <p className="mb-6 text-sm text-text-tertiary">
              An unexpected error occurred. Please try again.
            </p>
            <button
              className="rounded bg-gold px-4 py-2 text-sm font-medium text-bg-page transition-opacity hover:opacity-90"
              onClick={() => {
                this.setState({ hasError: false })
                this.props.onReset()
              }}
            >
              Go to Dashboard
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}

/** Route-aware ErrorBoundary (must be inside Router). */
export function ErrorBoundary({ children }: Props) {
  const navigate = useNavigate()
  return (
    <ErrorBoundaryInner onReset={() => navigate('/')}>
      {children}
    </ErrorBoundaryInner>
  )
}

/** Root-level ErrorBoundary that wraps the entire app (outside Router). */
export class RootErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[RootErrorBoundary]', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-screen flex-col items-center justify-center gap-4 bg-bg-page">
          <div className="text-center">
            <h1 className="mb-2 text-lg font-semibold text-text-primary">
              Something went wrong
            </h1>
            <p className="mb-6 text-sm text-text-tertiary">
              An unexpected error occurred. Please try again.
            </p>
            <button
              className="rounded bg-gold px-4 py-2 text-sm font-medium text-bg-page transition-opacity hover:opacity-90"
              onClick={() => {
                this.setState({ hasError: false })
                window.location.href = '/'
              }}
            >
              Go to Dashboard
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
