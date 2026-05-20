import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { useBranding } from '../contexts/BrandingContext'

export default function LoginPage() {
  const [msisdn, setMsisdn] = useState('')
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
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
      setError(err.response?.data?.error?.message || 'Login failed')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8">
        <div className="text-center mb-6">
          {branding?.logo_url && (
            <img src={branding.logo_url} alt="logo" className="h-16 mx-auto mb-4" />
          )}
          <h1 className="text-2xl font-bold text-gray-800">
            {branding?.product_name || 'SelfCare Portal'}
          </h1>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Phone Number</label>
            <input
              type="tel"
              value={msisdn}
              onChange={(e) => setMsisdn(e.target.value)}
              className="mt-1 w-full px-3 py-2 border rounded-md focus:ring-brand-primary focus:border-brand-primary"
              placeholder="79001234567"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">PIN Code</label>
            <input
              type="password"
              value={pin}
              onChange={(e) => setPin(e.target.value)}
              className="mt-1 w-full px-3 py-2 border rounded-md focus:ring-brand-primary focus:border-brand-primary"
              placeholder="****"
              maxLength={6}
              required
            />
          </div>
          {error && (
            <div className="text-red-600 text-sm">{error}</div>
          )}
          <button
            type="submit"
            disabled={isLoading}
            className="w-full bg-brand-primary text-white py-2 rounded-md hover:opacity-90 disabled:opacity-50"
          >
            {isLoading ? 'Loading...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  )
}
