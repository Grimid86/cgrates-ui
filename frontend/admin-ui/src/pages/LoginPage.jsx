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
      const status = err.response?.status
      const message = err.response?.data?.message || err.response?.data?.error?.message || 'Login failed'

      if (status === 403 && message.toLowerCase().includes('mfa')) {
        // Backend requires MFA but we haven't shown the input yet
        setStep('mfa')
        return
      }

      setError(message)
    }
  }

  const productName = branding?.product_name || 'Admin OSS'
  const primaryColor = branding?.colors?.primary || '#111827'

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-900">
      <div className="max-w-md w-full bg-white rounded-xl shadow-2xl p-8">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold" style={{ color: primaryColor }}>{productName}</h1>
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
              <label className="block text-sm font-medium text-gray-700">
                {mfaCode.length > 6 ? 'Backup Code' : 'TOTP Code'}
              </label>
              <input
                type="text"
                value={mfaCode}
                onChange={(e) => setMfaCode(e.target.value)}
                className="w-full px-4 py-2 border rounded-lg"
                placeholder="6-digit TOTP or 8-char backup code"
                required
              />
              <p className="text-xs text-gray-500 mt-1">
                Enter the 6-digit code from your authenticator app. If you lost access, use a backup code.
              </p>
              <button type="button" onClick={() => { setStep('credentials'); setMfaCode(''); setError('') }} className="text-sm text-blue-600 mt-2">
                ← Back to credentials
              </button>
            </div>
          )}
          <button
            type="submit"
            disabled={isLoading}
            className="w-full text-white py-2 rounded-lg hover:opacity-90 disabled:opacity-50"
            style={{ backgroundColor: primaryColor }}
          >
            {isLoading ? 'Authenticating...' : step === 'credentials' ? 'Continue' : 'Verify'}
          </button>
        </form>
      </div>
    </div>
  )
}
