import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import api from '../services/api'
import Card from '../components/ui/Card'

export default function TopUpSuccessPage() {
  const [balance, setBalance] = useState(null)
  const [loading, setLoading] = useState(true)

  const params = new URLSearchParams(window.location.search)
  const amount = params.get('amount') || '—'

  useEffect(() => {
    api.get('/balance')
      .then((res) => setBalance(res.data))
      .catch(() => setBalance(null))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="max-w-lg mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-green-700">✅ Top-Up Successful</h1>
      <Card>
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <span className="text-gray-600">Amount</span>
            <span className="text-xl font-bold">{amount} ₽</span>
          </div>
          {loading ? (
            <div className="text-sm text-gray-500">Updating balance...</div>
          ) : balance ? (
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Current Balance</span>
              <span className="text-xl font-bold text-blue-600">{balance.monetary?.toFixed(2)} ₽</span>
            </div>
          ) : (
            <div className="text-sm text-red-500">Could not load balance</div>
          )}
        </div>
      </Card>
      <div className="flex gap-4">
        <Link to="/" className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
          Go to Dashboard
        </Link>
        <Link to="/cdr" className="px-4 py-2 bg-gray-200 text-gray-800 rounded hover:bg-gray-300">
          View History
        </Link>
      </div>
    </div>
  )
}
