import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import api from '../services/api'
import { useI18n } from '../contexts/I18nContext'

export default function TenantsPage() {
  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [editId, setEditId] = useState(null)
  const [editName, setEditName] = useState('')
  const [editCode, setEditCode] = useState('')
  const queryClient = useQueryClient()
  const { t } = useI18n()

  const { data, isLoading } = useQuery('tenants', () => api.get('/tenants'))
  const tenants = data?.data?.data || []

  const createMutation = useMutation(
    (data) => api.post('/tenants', data),
    { onSuccess: () => { queryClient.invalidateQueries('tenants'); setShowForm(false); setName(''); setCode('') } }
  )

  const updateMutation = useMutation(
    ({ id, data }) => api.put(`/tenants/${id}`, data),
    { onSuccess: () => { queryClient.invalidateQueries('tenants'); setEditId(null) } }
  )

  const deleteMutation = useMutation(
    (id) => api.delete(`/tenants/${id}`),
    { onSuccess: () => queryClient.invalidateQueries('tenants') }
  )

  const handleSubmit = (e) => {
    e.preventDefault()
    createMutation.mutate({ name, code })
  }

  const handleUpdate = (e) => {
    e.preventDefault()
    updateMutation.mutate({ id: editId, data: { name: editName, code: editCode } })
  }

  const startEdit = (tenant) => {
    setEditId(tenant.id)
    setEditName(tenant.name)
    setEditCode(tenant.code)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t('nav.tenants', 'common')}</h1>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
          {showForm ? t('buttons.cancel', 'buttons') : t('buttons.create', 'buttons')}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Name</label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)} className="w-full px-4 py-2 border rounded-lg" required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Code</label>
            <input type="text" value={code} onChange={(e) => setCode(e.target.value)} className="w-full px-4 py-2 border rounded-lg" required />
          </div>
          <button type="submit" disabled={createMutation.isLoading} className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50">
            {createMutation.isLoading ? 'Creating...' : 'Create'}
          </button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50"><tr><th className="px-4 py-3 text-left">ID</th><th className="px-4 py-3 text-left">Name</th><th className="px-4 py-3 text-left">Code</th><th className="px-4 py-3 text-left">Status</th><th className="px-4 py-3 text-left">Actions</th></tr></thead>
          <tbody className="divide-y divide-gray-100">
            {tenants.map((t) => (
              <tr key={t.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm">{t.id}</td>
                <td className="px-4 py-3 text-sm font-medium">
                  {editId === t.id ? (
                    <input value={editName} onChange={(e) => setEditName(e.target.value)} className="border rounded px-2 py-1 w-full" />
                  ) : t.name}
                </td>
                <td className="px-4 py-3 text-sm">
                  {editId === t.id ? (
                    <input value={editCode} onChange={(e) => setEditCode(e.target.value)} className="border rounded px-2 py-1 w-full" />
                  ) : t.code}
                </td>
                <td className="px-4 py-3 text-sm">
                  <span className={`px-2 py-1 rounded text-xs ${t.is_active ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
                    {t.is_active ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm">
                  {editId === t.id ? (
                    <div className="flex gap-2">
                      <button onClick={handleUpdate} className="text-green-600 hover:underline">{t('buttons.save', 'buttons')}</button>
                      <button onClick={() => setEditId(null)} className="text-gray-500 hover:underline">{t('buttons.cancel', 'buttons')}</button>
                    </div>
                  ) : (
                    <div className="flex gap-2">
                      <button onClick={() => startEdit(t)} className="text-blue-600 hover:underline">{t('buttons.edit', 'buttons')}</button>
                      <button onClick={() => { if (confirm('Delete tenant?')) deleteMutation.mutate(t.id) }} className="text-red-600 hover:underline">{t('buttons.delete', 'buttons')}</button>
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
