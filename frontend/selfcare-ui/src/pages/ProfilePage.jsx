import { useState, useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import api from '../services/api'
import DataTable from '../components/ui/DataTable'

export default function ProfilePage() {
  const { user } = useAuth()
  const [profile, setProfile] = useState(null)
  const [email, setEmail] = useState('')
  const [oldPIN, setOldPIN] = useState('')
  const [newPIN, setNewPIN] = useState('')
  const [confirmPIN, setConfirmPIN] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [activeTab, setActiveTab] = useState('info')

  useEffect(() => {
    api.get('/profile').then(({ data }) => {
      setProfile(data)
      setEmail(data.email || '')
    })
  }, [])

  const handleUpdateProfile = async (e) => {
    e.preventDefault()
    setMessage('')
    setError('')
    try {
      await api.put('/profile', { email })
      setMessage('Profile updated successfully')
    } catch (err) {
      setError(err.response?.data?.error?.message || 'Update failed')
    }
  }

  const handleChangePIN = async (e) => {
    e.preventDefault()
    setMessage('')
    setError('')
    if (newPIN !== confirmPIN) {
      setError('New PINs do not match')
      return
    }
    try {
      await api.put('/profile/change-pin', { old_pin: oldPIN, new_pin: newPIN })
      setMessage('PIN changed successfully')
      setOldPIN('')
      setNewPIN('')
      setConfirmPIN('')
    } catch (err) {
      setError(err.response?.data?.error?.message || 'PIN change failed')
    }
  }

  const tabs = [
    { id: 'info', label: 'Personal Info' },
    { id: 'security', label: 'Security' },
    { id: 'sessions', label: 'Active Sessions' },
  ]

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">My Profile</h1>

      {/* Tabs */}
      <div className="bg-white rounded-t-lg shadow border-b">
        <div className="flex">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-6 py-3 text-sm font-medium border-b-2 ${
                activeTab === tab.id
                  ? 'border-brand-primary text-brand-primary'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-b-lg shadow p-6">
        {message && <div className="mb-4 p-3 bg-green-50 text-green-700 rounded">{message}</div>}
        {error && <div className="mb-4 p-3 bg-red-50 text-red-700 rounded">{error}</div>}

        {activeTab === 'info' && (
          <form onSubmit={handleUpdateProfile} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Phone Number</label>
              <input
                type="text"
                value={profile?.msisdn || user?.msisdn || ''}
                disabled
                className="mt-1 w-full px-3 py-2 border rounded-md bg-gray-50"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Category</label>
              <input
                type="text"
                value={profile?.category || 'prepaid'}
                disabled
                className="mt-1 w-full px-3 py-2 border rounded-md bg-gray-50"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="mt-1 w-full px-3 py-2 border rounded-md focus:ring-brand-primary focus:border-brand-primary"
                placeholder="your@email.com"
              />
            </div>
            <button
              type="submit"
              className="bg-brand-primary text-white px-4 py-2 rounded-md hover:opacity-90"
            >
              Save Changes
            </button>
          </form>
        )}

        {activeTab === 'security' && (
          <form onSubmit={handleChangePIN} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Current PIN</label>
              <input
                type="password"
                value={oldPIN}
                onChange={(e) => setOldPIN(e.target.value)}
                className="mt-1 w-full px-3 py-2 border rounded-md"
                placeholder="****"
                maxLength={6}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">New PIN</label>
              <input
                type="password"
                value={newPIN}
                onChange={(e) => setNewPIN(e.target.value)}
                className="mt-1 w-full px-3 py-2 border rounded-md"
                placeholder="****"
                maxLength={6}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Confirm New PIN</label>
              <input
                type="password"
                value={confirmPIN}
                onChange={(e) => setConfirmPIN(e.target.value)}
                className="mt-1 w-full px-3 py-2 border rounded-md"
                placeholder="****"
                maxLength={6}
                required
              />
            </div>
            <button
              type="submit"
              className="bg-brand-primary text-white px-4 py-2 rounded-md hover:opacity-90"
            >
              Change PIN
            </button>
          </form>
        )}

        {activeTab === 'sessions' && (
          <ActiveSessions />
        )}
      </div>
    </div>
  )
}

function ActiveSessions() {
  const { data, isLoading } = useQuery('sessions', () => api.get('/profile/sessions'))

  const columns = [
    { key: 'ip_address', title: 'IP Address' },
    { key: 'user_agent', title: 'Device', render: (v) => v?.split(' ')[0] || 'Unknown' },
    { key: 'issued_at', title: 'Started', render: (v) => new Date(v).toLocaleString() },
    { key: 'expires_at', title: 'Expires', render: (v) => new Date(v).toLocaleString() },
  ]

  return (
    <div>
      <p className="text-sm text-gray-500 mb-4">Manage your active sessions across devices.</p>
      <DataTable columns={columns} data={data?.data?.sessions || []} loading={isLoading} />
    </div>
  )
}
