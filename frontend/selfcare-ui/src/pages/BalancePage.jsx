import { useQuery } from 'react-query'
import api from '../services/api'
import DataTable from '../components/ui/DataTable'
import BalanceChart from '../components/charts/BalanceChart'

export default function BalancePage() {
  const { data, isLoading } = useQuery('balance-page', () => api.get('/balance'))

  const balances = data?.data?.balances || []

  const columns = [
    { key: 'type', title: 'Type', render: (v) => v.replace('*', '').charAt(0).toUpperCase() + v.replace('*', '').slice(1) },
    { key: 'value', title: 'Amount', render: (v, row) => `${v.toFixed(2)} ${row.currency || row.unit || ''}` },
    { key: 'expiry_date', title: 'Expires', render: (v) => v || 'No expiry' },
  ]

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">My Balance</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Balance Overview</h2>
          <BalanceChart balances={balances} />
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Details</h2>
          <DataTable columns={columns} data={balances} loading={isLoading} />
        </div>
      </div>
    </div>
  )
}
