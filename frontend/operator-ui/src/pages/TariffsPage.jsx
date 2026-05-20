import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import api from '../services/api'

export default function TariffsPage() {
  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [rate, setRate] = useState('')
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery('tariffs', () => api.get('/tariffs'))
  const tariffs = data?.data?.data || []

  const createMutation = useMutation(
    (data) => api.post('/tariffs', data),
    { onSuccess: () => { queryClient.invalidateQueries('tariffs'); setShowForm(false); setName(''); setRate('') } }
  )

  const handleSubmit = (e) => {
    e.preventDefault()
    createMutation.mutate({ name, rate: parseFloat(rate) })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Tariffs</h1>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
          {showForm ? 'Cancel' : 'Create Tariff'}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Name</label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)} className="w-full px-4 py-2 border rounded-lg" required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Rate</label>
            <input type="number" step="0.01" value={rate} onChange={(e) => setRate(e.target.value)} className="w-full px-4 py-2 border rounded-lg" required />
          </div>
          <button type="submit" disabled={createMutation.isLoading} className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50">
            {createMutation.isLoading ? 'Creating...' : 'Create'}
          </button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50"><tr><th className="px-4 py-3 text-left">ID</th><th className="px-4 py-3 text-left">Name</th><th className="px-4 py-3 text-left">Rate</th><th className="px-4 py-3 text-left">Created</th></tr></thead>
          <tbody className="divide-y divide-gray-100">
            {tariffs.map((t) => (
              <tr key={t.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm">{t.id}</td>
                <td className="px-4 py-3 text-sm font-medium">{t.name}</td>
                <td className="px-4 py-3 text-sm">₽ {t.rate}</td>
                <td className="px-4 py-3 text-sm text-gray-500">{new Date(t.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
