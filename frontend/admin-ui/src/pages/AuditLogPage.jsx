import { useState } from 'react'
import { useQuery } from 'react-query'
import api from '../services/api'

export default function AuditLogPage() {
  const [page, setPage] = useState(1)
  const [portalType, setPortalType] = useState('')

  const { data, isLoading } = useQuery(
    ['audit-log', page, portalType],
    () => {
      const params = new URLSearchParams()
      params.append('page', page)
      params.append('per_page', '20')
      if (portalType) params.append('portal_type', portalType)
      return api.get(`/audit-log?${params.toString()}`)
    },
    { keepPreviousData: true }
  )

  const logs = data?.data?.data || []
  const pagination = data?.data?.pagination || { page: 1, total_pages: 1, total: 0 }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Audit Log</h1>

      <div className="bg-white rounded-lg shadow p-4">
        <select value={portalType} onChange={(e) => { setPortalType(e.target.value); setPage(1) }} className="px-4 py-2 border rounded-lg">
          <option value="">All Portals</option>
          <option value="selfcare">SelfCare</option>
          <option value="operator">Operator</option>
          <option value="admin">Admin</option>
        </select>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr><th className="px-4 py-3 text-left">Time</th><th className="px-4 py-3 text-left">Portal</th><th className="px-4 py-3 text-left">Action</th><th className="px-4 py-3 text-left">Entity</th><th className="px-4 py-3 text-left">User</th></tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {logs.map((log, i) => (
              <tr key={i} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm text-gray-500">{new Date(log.created_at).toLocaleString()}</td>
                <td className="px-4 py-3 text-sm"><span className={`px-2 py-1 rounded text-xs ${
                  log.portal_type === 'admin' ? 'bg-red-100 text-red-700' : log.portal_type === 'operator' ? 'bg-blue-100 text-blue-700' : 'bg-green-100 text-green-700'
                }`}>{log.portal_type}</span></td>
                <td className="px-4 py-3 text-sm font-medium">{log.action}</td>
                <td className="px-4 py-3 text-sm">{log.entity_type} {log.entity_id}</td>
                <td className="px-4 py-3 text-sm">{log.user_id || 'System'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {pagination.total_pages > 1 && (
        <div className="flex justify-center gap-2">
          <button onClick={() => setPage(p => p - 1)} disabled={page === 1} className="px-3 py-1 border rounded bg-white disabled:opacity-50">Previous</button>
          <span className="px-3 py-1">{page} / {pagination.total_pages}</span>
          <button onClick={() => setPage(p => p + 1)} disabled={page === pagination.total_pages} className="px-3 py-1 border rounded bg-white disabled:opacity-50">Next</button>
        </div>
      )}
    </div>
  )
}
