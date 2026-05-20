import { useState } from 'react'
import { useQuery } from 'react-query'
import api from '../services/api'
import DataTable from '../components/ui/DataTable'
import Pagination from '../components/ui/Pagination'
import { useI18n } from '../contexts/I18nContext'

export default function SubscribersPage() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('')
  const { t } = useI18n()

  const { data, isLoading } = useQuery(
    ['subscribers', page, search, status],
    () => {
      const params = new URLSearchParams()
      params.append('page', page)
      params.append('per_page', '20')
      if (search) params.append('search', search)
      if (status) params.append('status', status)
      return api.get(`/subscribers?${params.toString()}`)
    },
    { keepPreviousData: true }
  )

  const subscribers = data?.data?.data || []
  const pagination = data?.data?.pagination || { page: 1, total_pages: 1, total: 0 }

  const columns = [
    { key: 'msisdn', title: 'MSISDN' },
    { key: 'imsi', title: 'IMSI' },
    { key: 'category', title: 'Category', render: (v) => v?.charAt(0).toUpperCase() + v?.slice(1) },
    {
      key: 'status',
      title: 'Status',
      render: (v) => (
        <span className={`px-2 py-1 rounded text-xs ${
          v === 'active' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
        }`}>{v}</span>
      ),
    },
    {
      key: 'actions',
      title: 'Actions',
      render: (_, row) => (
        <a href={`/subscribers/${row.id}`} className="text-blue-600 hover:underline text-sm">View</a>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t('nav.subscribers', 'common')}</h1>
        <a href="/subscribers/new" className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">{t('buttons.create', 'buttons')}</a>
      </div>

      <div className="bg-white rounded-lg shadow p-4 flex gap-4">
        <input
          type="text"
          placeholder="Search MSISDN, IMSI, email..."
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1) }}
          className="flex-1 px-4 py-2 border rounded-lg"
        />
        <select value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }} className="px-4 py-2 border rounded-lg">
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>
      </div>

      <DataTable columns={columns} data={subscribers} loading={isLoading} />
      <Pagination page={pagination.page} totalPages={pagination.total_pages} onPageChange={setPage} />
    </div>
  )
}
