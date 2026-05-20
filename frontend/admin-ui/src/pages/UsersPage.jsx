import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import api from '../services/api'

export default function UsersPage() {
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState(null)
  const [form, setForm] = useState({ email: '', role_code: '', password: '' })
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery('users', () => api.get('/users'))
  const users = data?.data?.data || []

  const createMutation = useMutation(
    (data) => api.post('/users', data),
    { onSuccess: () => { queryClient.invalidateQueries('users'); setShowForm(false); setForm({ email: '', role_code: '', password: '' }) } }
  )

  const updateMutation = useMutation(
    ({ id, data }) => api.put(`/users/${id}`, data),
    { onSuccess: () => { queryClient.invalidateQueries('users'); setEditId(null) } }
  )

  const deleteMutation = useMutation(
    (id) => api.delete(`/users/${id}`),
    { onSuccess: () => queryClient.invalidateQueries('users') }
  )

  const resetMFAMutation = useMutation(
    (id) => api.post(`/users/${id}/reset-mfa`),
    { onSuccess: () => alert('MFA reset successfully') }
  )

  const resetPasswordMutation = useMutation(
    (id) => api.post(`/users/${id}/reset-password`),
    { onSuccess: () => alert('Password reset email sent') }
  )

  const handleCreate = (e) => {
    e.preventDefault()
    createMutation.mutate(form)
  }

  const handleUpdate = (e, id) => {
    e.preventDefault()
    updateMutation.mutate({ id, data: { role_code: form.role_code, is_active: form.is_active } })
  }

  const startEdit = (u) => {
    setEditId(u.id)
    setForm({ email: u.email, role_code: u.role, is_active: u.is_active })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Staff Users</h1>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
          {showForm ? 'Cancel' : 'Create User'}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white rounded-lg shadow p-6 space-y-4">
          <div><label className="block text-sm font-medium text-gray-700">Email</label><input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} className="w-full px-4 py-2 border rounded-lg" required /></div>
          <div><label className="block text-sm font-medium text-gray-700">Role</label><input type="text" value={form.role_code} onChange={(e) => setForm({ ...form, role_code: e.target.value })} className="w-full px-4 py-2 border rounded-lg" required placeholder="admin" /></div>
          <div><label className="block text-sm font-medium text-gray-700">Password</label><input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} className="w-full px-4 py-2 border rounded-lg" required /></div>
          <button type="submit" disabled={createMutation.isLoading} className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50">
            {createMutation.isLoading ? 'Creating...' : 'Create'}
          </button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr><th className="px-4 py-3 text-left">ID</th><th className="px-4 py-3 text-left">Email</th><th className="px-4 py-3 text-left">Role</th><th className="px-4 py-3 text-left">Status</th><th className="px-4 py-3 text-left">Actions</th></tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {users.map((u) => (
              <tr key={u.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm">{u.id}</td>
                <td className="px-4 py-3 text-sm font-medium">{u.email}</td>
                <td className="px-4 py-3 text-sm">
                  {editId === u.id ? (
                    <input value={form.role_code} onChange={(e) => setForm({ ...form, role_code: e.target.value })} className="border rounded px-2 py-1 w-full" />
                  ) : (
                    <span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{u.role}</span>
                  )}
                </td>
                <td className="px-4 py-3 text-sm">
                  {editId === u.id ? (
                    <select value={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.value === 'true' })} className="border rounded px-2 py-1">
                      <option value="true">Active</option>
                      <option value="false">Inactive</option>
                    </select>
                  ) : (
                    <span className={`px-2 py-1 rounded text-xs ${u.is_active ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{u.is_active ? 'Active' : 'Inactive'}</span>
                  )}
                </td>
                <td className="px-4 py-3 text-sm">
                  {editId === u.id ? (
                    <div className="flex gap-2">
                      <button onClick={(e) => handleUpdate(e, u.id)} className="text-green-600 hover:underline">Save</button>
                      <button onClick={() => setEditId(null)} className="text-gray-500 hover:underline">Cancel</button>
                    </div>
                  ) : (
                    <div className="flex gap-2 flex-wrap">
                      <button onClick={() => startEdit(u)} className="text-blue-600 hover:underline">Edit</button>
                      <button onClick={() => resetMFAMutation.mutate(u.id)} className="text-orange-600 hover:underline">Reset MFA</button>
                      <button onClick={() => resetPasswordMutation.mutate(u.id)} className="text-purple-600 hover:underline">Reset PW</button>
                      <button onClick={() => { if (confirm('Delete user?')) deleteMutation.mutate(u.id) }} className="text-red-600 hover:underline">Delete</button>
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
