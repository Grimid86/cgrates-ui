import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import api from '../services/api'

export default function TariffsPage() {
  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [rate, setRate] = useState('')
  const [editId, setEditId] = useState(null)
  const [editName, setEditName] = useState('')
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery('tariffs', () => api.get('/tariffs'))
  const tariffs = data?.data?.data || []

  const createMutation = useMutation(
    (data) => api.post('/tariffs', data),
    { onSuccess: () => { queryClient.invalidateQueries('tariffs'); setShowForm(false); setName(''); setRate('') } }
  )

  const updateMutation = useMutation(
    ({ id, data }) => api.put(`/tariffs/${id}`, data),
    { onSuccess: () => { queryClient.invalidateQueries('tariffs'); setEditId(null) } }
  )

  const deleteMutation = useMutation(
    (id) => api.delete(`/tariffs/${id}`),
    { onSuccess: () => queryClient.invalidateQueries('tariffs') }
  )

  const activateMutation = useMutation(
    (id) => api.post(`/tariffs/${id}/activate`),
    { onSuccess: () => queryClient.invalidateQueries('tariffs') }
  )

  const handleSubmit = (e) => {
    e.preventDefault()
    createMutation.mutate({ name, config: { rate: parseFloat(rate) } })
  }

  const handleUpdate = (e) => {
    e.preventDefault()
    updateMutation.mutate({ id: editId, data: { name: editName } })
  }

  const startEdit = (t) => {
    setEditId(t.id)
    setEditName(t.name)
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
          <thead className="bg-gray-50"><tr><th className="px-4 py-3 text-left">ID</th><th className="px-4 py-3 text-left">Name</th><th className="px-4 py-3 text-left">Status</th><th className="px-4 py-3 text-left">Created</th><th className="px-4 py-3 text-left">Actions</th></tr></thead>
          <tbody className="divide-y divide-gray-100">
            {tariffs.map((t) => (
              <tr key={t.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm">{t.id}</td>
                <td className="px-4 py-3 text-sm font-medium">
                  {editId === t.id ? (
                    <input value={editName} onChange={(e) => setEditName(e.target.value)} className="border rounded px-2 py-1 w-full" />
                  ) : t.name}
                </td>
                <td className="px-4 py-3 text-sm">
                  <span className={`px-2 py-1 rounded text-xs ${t.status === 'active' ? 'bg-green-100 text-green-700' : t.status === 'draft' ? 'bg-gray-100 text-gray-700' : 'bg-red-100 text-red-700'}`}>
                    {t.status}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm text-gray-500">{new Date(t.created_at).toLocaleDateString()}</td>
                <td className="px-4 py-3 text-sm">
                  {editId === t.id ? (
                    <div className="flex gap-2">
                      <button onClick={handleUpdate} className="text-green-600 hover:underline">Save</button>
                      <button onClick={() => setEditId(null)} className="text-gray-500 hover:underline">Cancel</button>
                    </div>
                  ) : (
                    <div className="flex gap-2 flex-wrap">
                      <button onClick={() => startEdit(t)} className="text-blue-600 hover:underline">Edit</button>
                      {t.status !== 'active' && (
                        <button onClick={() => activateMutation.mutate(t.id)} className="text-green-600 hover:underline">Activate</button>
                      )}
                      <button onClick={() => { if (confirm('Delete tariff?')) deleteMutation.mutate(t.id) }} className="text-red-600 hover:underline">Delete</button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
