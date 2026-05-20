import { useQuery } from 'react-query'
import api from '../services/api'

export default function SessionsPage() {
  const { data, isLoading } = useQuery('active-sessions', () => api.get('/sessions/active'))
  const sessions = data?.data?.sessions || []

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Active Sessions</h1>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr><th className="px-4 py-3 text-left">MSISDN</th><th className="px-4 py-3 text-left">Origin Host</th><th className="px-4 py-3 text-left">Destination</th><th className="px-4 py-3 text-left">Duration</th><th className="px-4 py-3 text-left">Started</th></tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {sessions.map((s) => (
              <tr key={s.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm font-medium">{s.msisdn}</td>
                <td className="px-4 py-3 text-sm">{s.origin_host}</td>
                <td className="px-4 py-3 text-sm">{s.destination || 'N/A'}</td>
                <td className="px-4 py-3 text-sm">{s.duration || 0}s</td>
                <td className="px-4 py-3 text-sm text-gray-500">{new Date(s.start_time).toLocaleString()}</td>
              </tr>
            ))}
            {sessions.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No active sessions</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
