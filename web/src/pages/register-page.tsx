import { type FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuth } from '../hooks/use-auth'
import { ApiClientError } from '../api/client'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { FormField } from '../components/ui/form-field'
import { ROUTES } from '../lib/constants'
import type { ErrorDetail } from '../api/types'

const USERNAME_REGEX = /^[a-zA-Z0-9_]+$/

export function RegisterPage() {
  const { register } = useAuth()
  const navigate = useNavigate()

  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [generalError, setGeneralError] = useState('')

  const validate = (): boolean => {
    const errs: Record<string, string> = {}
    if (username.length < 3 || username.length > 50) {
      errs.username = 'Must be between 3 and 50 characters'
    } else if (!USERNAME_REGEX.test(username)) {
      errs.username = 'Only letters, numbers, and underscores allowed'
    }
    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      errs.email = 'Please enter a valid email address'
    }
    if (password.length < 8 || password.length > 72) {
      errs.password = 'Must be between 8 and 72 characters'
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
      await register({ username, email, password })
      navigate(ROUTES.LOGIN)
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

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-5">
      <p className="text-sm text-text-secondary">Create your account</p>

      {generalError ? (
        <p className="text-sm text-error">{generalError}</p>
      ) : null}

      <FormField label="Username" error={errors.username}>
        <Input
          type="text"
          placeholder="adventurer_01"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          error={!!errors.username}
        />
      </FormField>

      <FormField label="Email" error={errors.email}>
        <Input
          type="email"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={!!errors.email}
        />
      </FormField>

      <FormField label="Password" error={errors.password}>
        <Input
          type="password"
          placeholder="Min 8 characters"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          error={!!errors.password}
        />
      </FormField>

      <Button type="submit" loading={loading} className="mt-2 w-full">
        Create account
      </Button>

      <p className="text-center text-sm text-text-tertiary">
        Already have an account?{' '}
        <Link to={ROUTES.LOGIN} className="text-gold hover:text-gold-light">
          Sign in
        </Link>
      </p>
    </form>
  )
}
