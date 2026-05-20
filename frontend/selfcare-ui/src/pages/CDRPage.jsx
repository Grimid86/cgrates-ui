import { useState } from 'react'
import { useQuery } from 'react-query'
import api from '../services/api'
import { format } from 'date-fns'

export default function CDRPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading } = useQuery(['cdr', page], () => 
    api.get(`/cdr?page=${page}&per_page=20`)
  )

  if (isLoading) return <div>Loading...</div>

  const cdrs = data?.data?.data || []
  const pagination = data?.data?.pagination || {}

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Call History</h1>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left">Date</th>
              <th className="px-4 py-3 text-left">Type</th>
              <th className="px-4 py-3 text-left">Destination</th>
              <th className="px-4 py-3 text-left">Duration</th>
              <th className="px-4 py-3 text-left">Cost</th>
            </tr>
          </thead>
          <tbody>
            {cdrs.map((cdr, i) => (
              <tr key={i} className="border-t">
                <td className="px-4 py-3">
                  {format(new Date(cdr.started_at), 'dd.MM.yyyy HH:mm')}
                </td>
                <td className="px-4 py-3 capitalize">{cdr.type}</td>
                <td className="px-4 py-3">{cdr.destination}</td>
                <td className="px-4 py-3">{cdr.duration_seconds}s</td>
                <td className="px-4 py-3">{cdr.cost} {cdr.currency}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      
      {pagination.total_pages > 1 && (
        <div className="flex justify-center gap-2">
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
            className="px-4 py-2 bg-gray-200 rounded disabled:opacity-50"
          >
            Previous
          </button>
          <span className="px-4 py-2">{page} / {pagination.total_pages}</span>
          <button
            onClick={() => setPage(p => Math.min(pagination.total_pages, p + 1))}
            disabled={page === pagination.total_pages}
            className="px-4 py-2 bg-gray-200 rounded disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}
    </div>
  )
}
