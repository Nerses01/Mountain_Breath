import { useState } from 'react'
import { useNavigate } from 'react-router'
import { useLogin, useRegister } from '../api/hooks'
import { useFieldErrors } from '../i18n/useFieldErrors'

export function LoginPage() {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  // Controlled inputs: React state is the single source of truth for the
  // field values; the DOM just displays them.
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const navigate = useNavigate()
  const login = useLogin()
  const register = useRegister()
  const active = mode === 'login' ? login : register

  function onSubmit(e: React.FormEvent) {
    e.preventDefault() // stop the browser's own form POST + page reload
    active.mutate(
      { email, password },
      { onSuccess: () => navigate('/') },
    )
  }

  // The API answers with validation CODES, not sentences — this turns them
  // into readable text in the reader's language.
  const { fieldError, formError } = useFieldErrors(active.error)

  return (
    <div className="mx-auto max-w-sm px-4 py-12">
      <div className="rounded-xl border border-stone-200 bg-white p-6">
        <div className="flex gap-2">
          <ModeTab label="Sign in" active={mode === 'login'} onClick={() => setMode('login')} />
          <ModeTab label="Create account" active={mode === 'register'} onClick={() => setMode('register')} />
        </div>

        <form onSubmit={onSubmit} className="mt-6 space-y-4">
          <Field
            label="Email"
            type="email"
            value={email}
            onChange={setEmail}
            error={fieldError('email')}
          />
          <Field
            label="Password"
            type="password"
            value={password}
            onChange={setPassword}
            error={fieldError('password')}
          />

          {formError && (
            <p className="rounded-lg bg-red-50 p-3 text-sm text-red-600">{formError}</p>
          )}

          <button
            type="submit"
            disabled={active.isPending}
            className="w-full rounded-lg bg-emerald-700 py-2.5 font-medium text-white hover:bg-emerald-800 disabled:opacity-50"
          >
            {active.isPending
              ? 'Please wait…'
              : mode === 'login'
                ? 'Sign in'
                : 'Create account'}
          </button>
        </form>
      </div>
    </div>
  )
}

function ModeTab({
  label,
  active,
  onClick,
}: {
  label: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        active
          ? 'flex-1 rounded-lg bg-stone-800 py-2 text-sm font-medium text-white'
          : 'flex-1 rounded-lg py-2 text-sm font-medium text-stone-500 hover:bg-stone-100'
      }
    >
      {label}
    </button>
  )
}

function Field({
  label,
  type,
  value,
  onChange,
  error,
}: {
  label: string
  type: string
  value: string
  onChange: (v: string) => void
  error?: string
}) {
  return (
    <label className="block">
      <span className="text-sm font-medium text-stone-600">{label}</span>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        required
        className="mt-1 w-full rounded-lg border border-stone-300 px-3 py-2 text-sm focus:border-emerald-600 focus:outline-none"
      />
      {error && <span className="mt-1 block text-xs text-red-600">{error}</span>}
    </label>
  )
}
