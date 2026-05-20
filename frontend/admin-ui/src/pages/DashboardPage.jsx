import { useQuery } from 'react-query'
import api from '../services/api'

export default function DashboardPage() {
  const { data: tenantsData } = useQuery('tenants-count', () => api.get('/tenants?per_page=1'))
  const { data: usersData } = useQuery('users-count', () => api.get('/users?per_page=1'))
  const { data: healthData } = useQuery('health', () => api.get('/health'))

  const totalTenants = tenantsData?.data?.data?.length || 0
  const totalUsers = usersData?.data?.data?.length || 0

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">System Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-5 border-l-4 border-blue-500">
          <p className="text-sm text-gray-500">Tenants</p>
          <p className="text-2xl font-bold">{totalTenants}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-5 border-l-4 border-green-500">
          <p className="text-sm text-gray-500">Staff Users</p>
          <p className="text-2xl font-bold">{totalUsers}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-5 border-l-4 border-yellow-500">
          <p className="text-sm text-gray-500">System Status</p>
          <p className="text-2xl font-bold text-green-600">{healthData?.data?.status || 'Unknown'}</p>
        </div>
        <div className="bg-white rounded-lg shadow p-5 border-l-4 border-red-500">
          <p className="text-sm text-gray-500">Version</p>
          <p className="text-2xl font-bold">{healthData?.data?.version || '1.0.0'}</p>
        </div>
      </div>
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
        <div className="flex gap-3 flex-wrap">
          <a href="/tenants" className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Manage Tenants</a>
          <a href="/users" className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700">Manage Users</a>
          <a href="/rbac" className="px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700">RBAC Matrix</a>
          <a href="/monitoring" className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700">Monitoring</a>
        </div>
      </div>
    </div>
  )
}
