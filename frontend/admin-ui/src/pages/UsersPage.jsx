import { useQuery } from 'react-query'
import api from '../services/api'

export default function UsersPage() {
  const { data, isLoading } = useQuery('users', () => api.get('/users'))
  const users = data?.data?.data || []

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Staff Users</h1>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr><th className="px-4 py-3 text-left">ID</th><th className="px-4 py-3 text-left">Email</th><th className="px-4 py-3 text-left">Role</th><th className="px-4 py-3 text-left">Tenant</th><th className="px-4 py-3 text-left">Status</th></tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {users.map((u) => (
              <tr key={u.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm">{u.id}</td>
                <td className="px-4 py-3 text-sm font-medium">{u.email}</td>
                <td className="px-4 py-3 text-sm"><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{u.role}</span></td>
                <td className="px-4 py-3 text-sm">{u.tenant_id}</td>
                <td className="px-4 py-3 text-sm"><span className={`px-2 py-1 rounded text-xs ${u.is_active ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{u.is_active ? 'Active' : 'Inactive'}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
