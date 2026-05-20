import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { useBranding } from '../contexts/BrandingContext'

export default function LoginPage() {
  const [msisdn, setMsisdn] = useState('')
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
  const [attempts, setAttempts] = useState(0)
  const { login, isLoading } = useAuth()
  const { branding } = useBranding()
  const navigate = useNavigate()

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    try {
      await login(msisdn, pin)
      navigate('/')
    } catch (err) {
      setAttempts((a) => a + 1)
      setError(err.response?.data?.error?.message || 'Invalid credentials')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100"
      style={branding?.login_background_url ? {
        backgroundImage: `url(${branding.login_background_url})`,
        backgroundSize: 'cover',
        backgroundPosition: 'center',
      } : {}}
    >
      <div className="max-w-md w-full bg-white/95 backdrop-blur rounded-xl shadow-2xl p-8">
        <div className="text-center mb-8">
          {branding?.logo_url && (
            <img src={branding.logo_url} alt="logo" className="h-16 mx-auto mb-4 object-contain" />
          )}
          <h1 className="text-2xl font-bold text-gray-800">
            {branding?.product_name || 'SelfCare Portal'}
          </h1>
          <p className="text-gray-500 mt-1">Sign in to your account</p>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded-lg text-sm">
            {error}
            {attempts >= 3 && (
              <p className="mt-1 text-xs">Too many attempts. Your account may be locked.</p>
            )}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Phone Number</label>
            <input
              type="tel"
              value={msisdn}
              onChange={(e) => setMsisdn(e.target.value.replace(/\D/g, ''))}
              className="w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary transition"
              placeholder="79001234567"
              maxLength={15}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">PIN Code</label>
            <input
              type="password"
              value={pin}
              onChange={(e) => setPin(e.target.value.replace(/\D/g, ''))}
              className="w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-brand-primary focus:border-brand-primary transition"
              placeholder="••••"
              maxLength={6}
              required
            />
          </div>
          <button
            type="submit"
            disabled={isLoading}
            className="w-full bg-brand-primary text-white py-3 rounded-lg hover:opacity-90 disabled:opacity-50 font-semibold transition"
          >
            {isLoading ? (
              <span className="flex items-center justify-center">
                <svg className="animate-spin h-5 w-5 mr-2" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                Signing in...
              </span>
            ) : 'Sign In'}
          </button>
        </form>

        <div className="mt-6 text-center text-sm text-gray-500">
          <p>Need help? Contact support</p>
          {branding?.support_email && (
            <a href={`mailto:${branding.support_email}`} className="text-brand-primary hover:underline">
              {branding.support_email}
            </a>
          )}
        </div>
      </div>
    </div>
  )
}
