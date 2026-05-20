import { useQuery } from 'react-query'
import api from '../services/api'

export default function MonitoringPage() {
  const { data: dbData } = useQuery('db-status', () => api.get('/database/status'))
  const { data: pulsarData } = useQuery('pulsar-lag', () => api.get('/pulsar/lag'))

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">System Monitoring</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Database Status</h2>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span>Primary:</span> <span className="text-green-600 font-medium">{dbData?.data?.primary?.status || 'Unknown'}</span></div>
            <div className="flex justify-between"><span>Replication Lag:</span> <span>{dbData?.data?.primary?.lag_seconds || 0}s</span></div>
            <div className="flex justify-between"><span>Vacuum:</span> <span>{dbData?.data?.vacuum_status || 'Unknown'}</span></div>
            <div className="mt-4">
              <p className="text-gray-500 mb-2">Partitions:</p>
              <div className="flex flex-wrap gap-2">
                {(dbData?.data?.partitions || []).map((p) => (
                  <span key={p} className="px-2 py-1 bg-gray-100 rounded text-xs">{p}</span>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Pulsar Lag</h2>
          <div className="space-y-2 text-sm">
            {Object.entries(pulsarData?.data?.lag || {}).map(([topic, lag]) => (
              <div key={topic} className="flex justify-between items-center">
                <span className="truncate max-w-xs">{topic}</span>
                <span className={`font-medium ${lag > 1000 ? 'text-red-600' : 'text-green-600'}`}>{lag}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
