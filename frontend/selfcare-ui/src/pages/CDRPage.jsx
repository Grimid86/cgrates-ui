import { useState } from 'react'
import { useQuery } from 'react-query'
import api from '../services/api'
import { format } from 'date-fns'
import DataTable from '../components/ui/DataTable'
import Pagination from '../components/ui/Pagination'

export default function CDRPage() {
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState({ from: '', to: '', type: '' })

  const { data, isLoading } = useQuery(
    ['cdr', page, filters],
    () => {
      const params = new URLSearchParams()
      params.append('page', page)
      params.append('per_page', '20')
      if (filters.from) params.append('from', filters.from)
      if (filters.to) params.append('to', filters.to)
      if (filters.type) params.append('type', filters.type)
      return api.get(`/cdr?${params.toString()}`)
    },
    { keepPreviousData: true }
  )

  const cdrs = data?.data?.data || []
  const pagination = data?.data?.pagination || { page: 1, per_page: 20, total: 0, total_pages: 1 }

  const columns = [
    {
      key: 'started_at',
      title: 'Date',
      render: (v) => format(new Date(v), 'dd.MM.yyyy HH:mm'),
    },
    { key: 'type', title: 'Type', render: (v) => v?.charAt(0).toUpperCase() + v?.slice(1) },
    { key: 'destination', title: 'Destination' },
    {
      key: 'duration_seconds',
      title: 'Duration',
      render: (v) => {
        if (!v) return '-'
        const mins = Math.floor(v / 60)
        const secs = v % 60
        return `${mins}:${secs.toString().padStart(2, '0')}`
      },
    },
    { key: 'cost', title: 'Cost', render: (v, row) => `${v?.toFixed(2)} ${row.currency || '₽'}` },
    {
      key: 'status',
      title: 'Status',
      render: (v) => (
        <span className={`px-2 py-1 rounded text-xs ${
          v === 'completed' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'
        }`}>
          {v}
        </span>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Call History</h1>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow p-4">
        <div className="flex flex-wrap gap-4">
          <div>
            <label className="block text-sm text-gray-600 mb-1">From</label>
            <input
              type="date"
              value={filters.from}
              onChange={(e) => setFilters((f) => ({ ...f, from: e.target.value }))}
              className="border rounded px-3 py-2"
            />
          </div>
          <div>
            <label className="block text-sm text-gray-600 mb-1">To</label>
            <input
              type="date"
              value={filters.to}
              onChange={(e) => setFilters((f) => ({ ...f, to: e.target.value }))}
              className="border rounded px-3 py-2"
            />
          </div>
          <div>
            <label className="block text-sm text-gray-600 mb-1">Type</label>
            <select
              value={filters.type}
              onChange={(e) => setFilters((f) => ({ ...f, type: e.target.value }))}
              className="border rounded px-3 py-2"
            >
              <option value="">All</option>
              <option value="voice">Voice</option>
              <option value="data">Data</option>
              <option value="sms">SMS</option>
            </select>
          </div>
          <div className="flex items-end">
            <button
              onClick={() => { setFilters({ from: '', to: '', type: '' }); setPage(1) }}
              className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800"
            >
              Reset
            </button>
          </div>
        </div>
      </div>

      <DataTable columns={columns} data={cdrs} loading={isLoading} />
      <Pagination
        page={pagination.page}
        totalPages={pagination.total_pages}
        total={pagination.total}
        onPageChange={setPage}
      />
    </div>
  )
}
