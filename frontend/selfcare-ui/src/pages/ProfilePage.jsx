import { useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import api from '../services/api'

export default function ProfilePage() {
  const { user } = useAuth()
  const [email, setEmail] = useState('')
  const [message, setMessage] = useState('')

  const handleUpdate = async (e) => {
    e.preventDefault()
    try {
      await api.put('/profile', { email })
      setMessage('Profile updated successfully')
    } catch (err) {
      setMessage(err.response?.data?.error?.message || 'Update failed')
    }
  }

  return (
    <div className="max-w-xl mx-auto bg-white rounded-lg shadow p-6">
      <h1 className="text-2xl font-bold mb-6">My Profile</h1>
      
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">Phone Number</label>
          <p className="mt-1 text-gray-900">{user?.msisdn}</p>
        </div>
        
        <form onSubmit={handleUpdate} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 w-full px-3 py-2 border rounded-md"
              placeholder="your@email.com"
            />
          </div>
          
          {message && (
            <div className="text-sm text-green-600">{message}</div>
          )}
          
          <button
            type="submit"
            className="bg-brand-primary text-white px-4 py-2 rounded-md hover:opacity-90"
          >
            Save Changes
          </button>
        </form>
      </div>
    </div>
  )
}
