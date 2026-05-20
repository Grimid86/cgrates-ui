import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { useBranding } from '../contexts/BrandingContext'

export default function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [mfaCode, setMfaCode] = useState('')
  const [step, setStep] = useState('credentials')
  const [error, setError] = useState('')
  const { login, isLoading } = useAuth()
  const { branding } = useBranding()
  const navigate = useNavigate()

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      const data = await login(email, password, mfaCode)
      if (data.mfa_required && step === 'credentials') {
        setStep('mfa')
        return
      }
      navigate('/')
    } catch (err) {
      setError(err.response?.data?.error?.message || 'Login failed')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-900">
      <div className="max-w-md w-full bg-white rounded-xl shadow-2xl p-8">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-gray-900">Admin OSS</h1>
          <p className="text-red-600 mt-1 font-medium">🔒 Restricted Access</p>
        </div>

        {error && <div className="mb-4 p-3 bg-red-50 text-red-700 rounded text-sm">{error}</div>}

        <form onSubmit={handleSubmit} className="space-y-4">
          {step === 'credentials' ? (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700">Email</label>
                <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} className="w-full px-4 py-2 border rounded-lg" required />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">Password</label>
                <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="w-full px-4 py-2 border rounded-lg" required />
              </div>
            </>
          ) : (
            <div>
              <label className="block text-sm font-medium text-gray-700">TOTP Code</label>
              <input type="text" value={mfaCode} onChange={(e) => setMfaCode(e.target.value)} className="w-full px-4 py-2 border rounded-lg" maxLength={6} required />
              <button type="button" onClick={() => setStep('credentials')} className="text-sm text-blue-600 mt-2">Back</button>
            </div>
          )}
          <button type="submit" disabled={isLoading} className="w-full bg-gray-900 text-white py-2 rounded-lg hover:bg-gray-800 disabled:opacity-50">
            {isLoading ? 'Authenticating...' : step === 'credentials' ? 'Continue' : 'Verify'}
          </button>
        </form>
      </div>
    </div>
  )
}
