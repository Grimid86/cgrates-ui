import { useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import api from '../services/api'

export default function ProfilePage() {
  const [mfaData, setMfaData] = useState(null)
  const [verifyCode, setVerifyCode] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSetupMFA = async () => {
    setError('')
    setSuccess('')
    setLoading(true)
    try {
      const { data } = await api.post('/auth/mfa/setup')
      setMfaData(data)
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to setup MFA')
    } finally {
      setLoading(false)
    }
  }

  const handleVerifyMFA = async (e) => {
    e.preventDefault()
    setError('')
    setSuccess('')
    try {
      await api.post('/auth/mfa/verify', { code: verifyCode })
      setSuccess('MFA enabled successfully!')
      setMfaData(null)
      setVerifyCode('')
    } catch (err) {
      setError(err.response?.data?.message || 'Verification failed')
    }
  }

  const handleDisableMFA = async () => {
    if (!confirm('Are you sure? This will remove MFA protection from your account.')) return
    setError('')
    setSuccess('')
    try {
      await api.post('/auth/mfa/disable')
      setSuccess('MFA disabled.')
      setMfaData(null)
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to disable MFA')
    }
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">My Profile</h1>

      {error && <div className="p-3 bg-red-50 text-red-700 rounded text-sm">{error}</div>}
      {success && <div className="p-3 bg-green-50 text-green-700 rounded text-sm">{success}</div>}

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Multi-Factor Authentication (MFA)</h2>

        {!mfaData ? (
          <div className="space-y-3">
            <p className="text-sm text-gray-600">
              Protect your account with TOTP-based two-factor authentication using an authenticator app
              (Google Authenticator, Authy, Microsoft Authenticator).
            </p>
            <div className="flex gap-3">
              <button
                onClick={handleSetupMFA}
                disabled={loading}
                className="px-4 py-2 bg-brand-primary text-white rounded-lg hover:opacity-90 disabled:opacity-50 text-sm"
              >
                {loading ? 'Setting up...' : 'Enable MFA'}
              </button>
              <button
                onClick={handleDisableMFA}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 text-sm"
              >
                Disable MFA
              </button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex flex-col items-center bg-gray-50 p-4 rounded-lg">
              <p className="text-sm font-medium text-gray-700 mb-2">Scan this QR code with your authenticator app</p>
              <QRCodeSVG value={mfaData.qr_code_url} size={200} />
              <p className="text-xs text-gray-500 mt-2 break-all text-center max-w-xs">{mfaData.qr_code_url}</p>
            </div>

            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <h3 className="text-sm font-semibold text-yellow-800 mb-2">🔐 Backup Codes (save these!)</h3>
              <p className="text-xs text-yellow-700 mb-2">Each code can only be used once. Store them securely.</p>
              <div className="grid grid-cols-4 gap-2">
                {mfaData.backup_codes.map((code) => (
                  <code key={code} className="text-xs bg-white border border-yellow-300 rounded px-2 py-1 text-center font-mono">{code}</code>
                ))}
              </div>
            </div>

            <form onSubmit={handleVerifyMFA} className="space-y-3">
              <p className="text-sm text-gray-600">Enter the 6-digit code from your authenticator app to activate MFA:</p>
              <input
                type="text"
                value={verifyCode}
                onChange={(e) => setVerifyCode(e.target.value)}
                placeholder="123456"
                maxLength={8}
                className="w-full px-4 py-2 border rounded-lg"
                required
              />
              <div className="flex gap-3">
                <button type="submit" className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm">
                  Verify & Enable
                </button>
                <button type="button" onClick={() => setMfaData(null)} className="px-4 py-2 border text-gray-700 rounded-lg hover:bg-gray-50 text-sm">
                  Cancel
                </button>
              </div>
            </form>
          </div>
        )}
      </div>
    </div>
  )
}
