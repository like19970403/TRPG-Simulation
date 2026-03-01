import { type FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useAuth } from '../hooks/use-auth'
import { ApiClientError } from '../api/client'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { FormField } from '../components/ui/form-field'
import { ROUTES } from '../lib/constants'
import type { ErrorDetail } from '../api/types'

export function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [generalError, setGeneralError] = useState('')

  const validate = (): boolean => {
    const errs: Record<string, string> = {}
    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      errs.email = '請輸入有效的電子郵件地址'
    }
    if (!password) {
      errs.password = '請輸入密碼'
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
      await login({ email, password })
      navigate(ROUTES.DASHBOARD)
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

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-5">
      <p className="text-sm text-text-secondary">登入你的帳號</p>

      {generalError ? (
        <p className="text-sm text-error">{generalError}</p>
      ) : null}

      <FormField label="電子郵件" error={errors.email}>
        <Input
          type="email"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={!!errors.email}
        />
      </FormField>

      <FormField label="密碼" error={errors.password}>
        <Input
          type="password"
          placeholder="••••••••"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          error={!!errors.password}
        />
      </FormField>

      <Button type="submit" loading={loading} className="mt-2 w-full">
        登入
      </Button>

      <p className="text-center text-sm text-text-tertiary">
        還沒有帳號？{' '}
        <Link to={ROUTES.REGISTER} className="text-gold hover:text-gold-light">
          註冊帳號
        </Link>
      </p>
    </form>
  )
}
