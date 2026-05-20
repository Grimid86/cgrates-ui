import { useParams } from 'react-router-dom'
import { useQuery } from 'react-query'
import api from '../services/api'
import Card from '../components/ui/Card'

export default function SubscriberDetailPage() {
  const { id } = useParams()
  const { data, isLoading } = useQuery(['subscriber', id], () => api.get(`/subscribers/${id}`))

  if (isLoading) return <div className="text-center py-8">Loading...</div>

  const sub = data?.data || {}

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Subscriber {sub.msisdn}</h1>
        <div className="flex gap-2">
          <button className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700">Adjust Balance</button>
          <button className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700">Block</button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card title="Status" value={sub.status} color={sub.status === 'active' ? 'success' : 'danger'} />
        <Card title="Category" value={sub.category} color="primary" />
        <Card title="IMSI" value={sub.imsi || 'N/A'} color="warning" />
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Account Information</h2>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div><span className="text-gray-500">ID:</span> {sub.id}</div>
          <div><span className="text-gray-500">MSISDN:</span> {sub.msisdn}</div>
          <div><span className="text-gray-500">Email:</span> {sub.email || 'N/A'}</div>
          <div><span className="text-gray-500">Created:</span> {sub.created_at ? new Date(sub.created_at).toLocaleString() : 'N/A'}</div>
        </div>
      </div>
    </div>
  )
}
