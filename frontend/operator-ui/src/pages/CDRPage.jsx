import { useState } from 'react'
import { useQuery } from 'react-query'
import api from '../services/api'
import DataTable from '../components/ui/DataTable'
import Pagination from '../components/ui/Pagination'

export default function CDRPage() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')

  const { data, isLoading } = useQuery(
    ['cdrs', page, search],
    () => {
      const params = new URLSearchParams()
      params.append('page', page)
      params.append('per_page', '20')
      if (search) params.append('search', search)
      return api.get(`/cdr?${params.toString()}`)
    },
    { keepPreviousData: true }
  )

  const cdrs = data?.data?.data || []
  const pagination = data?.data?.pagination || { page: 1, total_pages: 1, total: 0 }

  const columns = [
    { key: 'callid', title: 'Call ID' },
    { key: 'msisdn', title: 'MSISDN' },
    { key: 'destination', title: 'Destination' },
    { key: 'duration', title: 'Duration (s)' },
    { key: 'cost', title: 'Cost', render: (v) => `₽ ${v}` },
    { key: 'created_at', title: 'Date', render: (v) => new Date(v).toLocaleString() },
  ]

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">CDR Records</h1>
      <div className="bg-white rounded-lg shadow p-4">
        <input
          type="text"
          placeholder="Search MSISDN, destination..."
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1) }}
          className="w-full px-4 py-2 border rounded-lg"
        />
      </div>
      <DataTable columns={columns} data={cdrs} loading={isLoading} />
      <Pagination page={pagination.page} totalPages={pagination.total_pages} onPageChange={setPage} />
    </div>
  )
}
