import { useQuery } from 'react-query'
import api from '../services/api'
import Card from '../components/ui/Card'

export default function DashboardPage() {
  const { data: subscribersData } = useQuery('subscribers-count', () => api.get('/subscribers?per_page=1'))
  const { data: sessionsData } = useQuery('sessions-count', () => api.get('/sessions/active'))
  const { data: cdrData } = useQuery('cdr-today', () => api.get('/cdr?from=2026-05-20&per_page=1'))

  const totalSubscribers = subscribersData?.data?.pagination?.total || 0
  const activeSessions = sessionsData?.data?.total_active || 0
  const totalCDRs = cdrData?.data?.pagination?.total || 0

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card title="Total Subscribers" value={totalSubscribers.toLocaleString()} color="primary" />
        <Card title="Active Sessions" value={activeSessions.toLocaleString()} color="success" />
        <Card title="CDRs Today" value={totalCDRs.toLocaleString()} color="warning" />
        <Card title="Revenue Today" value="₽ 0.00" color="danger" />
      </div>
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
        <div className="flex gap-3">
          <a href="/subscribers" className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Manage Subscribers</a>
          <a href="/tariffs" className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700">Manage Tariffs</a>
          <a href="/cdr" className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700">View CDRs</a>
        </div>
      </div>
    </div>
  )
}
