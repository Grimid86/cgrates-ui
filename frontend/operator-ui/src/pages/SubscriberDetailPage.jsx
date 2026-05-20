import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import api from '../services/api'
import Card from '../components/ui/Card'

export default function SubscriberDetailPage() {
  const { id } = useParams()
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery(['subscriber', id], () => api.get(`/subscribers/${id}`))
  const sub = data?.data || {}

  const blockMutation = useMutation(() => api.post(`/subscribers/${id}/block`), {
    onSuccess: () => queryClient.invalidateQueries(['subscriber', id]),
  })
  const unblockMutation = useMutation(() => api.post(`/subscribers/${id}/unblock`), {
    onSuccess: () => queryClient.invalidateQueries(['subscriber', id]),
  })
  const freezeMutation = useMutation(() => api.post(`/subscribers/${id}/balance/freeze`), {
    onSuccess: () => queryClient.invalidateQueries(['subscriber', id]),
  })
  const unfreezeMutation = useMutation(() => api.post(`/subscribers/${id}/balance/unfreeze`), {
    onSuccess: () => queryClient.invalidateQueries(['subscriber', id]),
  })

  if (isLoading) return <div className="text-center py-8">Loading...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Subscriber {sub.msisdn}</h1>
        <div className="flex gap-2 flex-wrap">
          {sub.status === 'active' ? (
            <button onClick={() => blockMutation.mutate()} className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700">Block</button>
          ) : (
            <button onClick={() => unblockMutation.mutate()} className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700">Unblock</button>
          )}
          {!sub.balance_frozen_at ? (
            <button onClick={() => freezeMutation.mutate()} className="px-4 py-2 bg-orange-600 text-white rounded hover:bg-orange-700">Freeze Balance</button>
          ) : (
            <button onClick={() => unfreezeMutation.mutate()} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Unfreeze Balance</button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card title="Status" value={sub.status} color={sub.status === 'active' ? 'success' : 'danger'} />
        <Card title="Category" value={sub.category} color="primary" />
        <Card title="Balance" value={`${sub.balance?.toFixed?.(2) || sub.balance || 0} ₽`} color="warning" />
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Account Information</h2>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div><span className="text-gray-500">ID:</span> {sub.id}</div>
          <div><span className="text-gray-500">MSISDN:</span> {sub.msisdn}</div>
          <div><span className="text-gray-500">IMSI:</span> {sub.imsi || 'N/A'}</div>
          <div><span className="text-gray-500">Email:</span> {sub.email || 'N/A'}</div>
          <div><span className="text-gray-500">Created:</span> {sub.created_at ? new Date(sub.created_at).toLocaleString() : 'N/A'}</div>
          <div><span className="text-gray-500">Balance Frozen:</span> {sub.balance_frozen_at ? new Date(sub.balance_frozen_at).toLocaleString() : 'No'}</div>
        </div>
      </div>
    </div>
  )
}
