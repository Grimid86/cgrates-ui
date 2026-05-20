import { useQuery } from 'react-query'
import api from '../services/api'

export default function BalancePage() {
  const { data, isLoading } = useQuery('balance', () => api.get('/balance'))

  if (isLoading) return <div>Loading...</div>

  const balances = data?.data?.balances || []

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">My Balance</h1>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left">Type</th>
              <th className="px-4 py-3 text-left">Amount</th>
              <th className="px-4 py-3 text-left">Expires</th>
            </tr>
          </thead>
          <tbody>
            {balances.map((b, i) => (
              <tr key={i} className="border-t">
                <td className="px-4 py-3">{b.type}</td>
                <td className="px-4 py-3 font-medium">
                  {b.value} {b.currency || b.unit}
                </td>
                <td className="px-4 py-3 text-gray-500">
                  {b.expiry_date || 'No expiry'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
