import { RouterProvider } from 'react-router'
import { router } from './router'
import { RootErrorBoundary } from './components/error-boundary'
import { ToastContainer } from './components/toast-container'

export default function App() {
  return (
    <RootErrorBoundary>
      <RouterProvider router={router} />
      <ToastContainer />
    </RootErrorBoundary>
  )
}
