import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../services/api'

export default function SubscriberNewPage() {
  const navigate = useNavigate()
  const [form, setForm] = useState({
    msisdn: '',
    imsi: '',
    email: '',
    category: 'prepaid',
    pin: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleChange = (e) => {
    setForm((prev) => ({ ...prev, [e.target.name]: e.target.value }))
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const payload = {
        msisdn: form.msisdn,
        pin: form.pin,
        category: form.category,
      }
      if (form.imsi) payload.imsi = form.imsi
      if (form.email) payload.email = form.email

      const { data } = await api.post('/subscribers', payload)
      navigate(`/subscribers/${data.id}`)
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to create subscriber')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Add Subscriber</h1>
      {error && <div className="mb-4 p-3 bg-red-100 text-red-700 rounded">{error}</div>}
      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">MSISDN *</label>
          <input
            name="msisdn"
            value={form.msisdn}
            onChange={handleChange}
            required
            pattern="[0-9]+"
            className="w-full border rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="79161234567"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">IMSI</label>
          <input
            name="imsi"
            value={form.imsi}
            onChange={handleChange}
            className="w-full border rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="250015555555555"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
          <input
            name="email"
            type="email"
            value={form.email}
            onChange={handleChange}
            className="w-full border rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="subscriber@example.com"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Category *</label>
          <select
            name="category"
            value={form.category}
            onChange={handleChange}
            className="w-full border rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="prepaid">Prepaid</option>
            <option value="postpaid">Postpaid</option>
            <option value="enterprise">Enterprise</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">PIN *</label>
          <input
            name="pin"
            type="password"
            value={form.pin}
            onChange={handleChange}
            required
            minLength={4}
            maxLength={6}
            className="w-full border rounded px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="1234"
          />
        </div>
        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? 'Creating...' : 'Create Subscriber'}
          </button>
          <button
            type="button"
            onClick={() => navigate('/subscribers')}
            className="px-4 py-2 bg-gray-200 text-gray-800 rounded hover:bg-gray-300"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}
