import { useState } from 'react'
import { useQuery } from 'react-query'
import api from '../services/api'

export default function ReportsPage() {
  const [reportType, setReportType] = useState('revenue')
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')

  const { data, isLoading } = useQuery(
    ['report', reportType, fromDate, toDate],
    () => api.get(`/reports/${reportType}?from=${fromDate}&to=${toDate}`),
    { enabled: !!fromDate && !!toDate }
  )

  const reportData = data?.data || []

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Reports</h1>

      <div className="bg-white rounded-lg shadow p-4 flex gap-4 flex-wrap items-end">
        <div>
          <label className="block text-sm font-medium text-gray-700">Report Type</label>
          <select value={reportType} onChange={(e) => setReportType(e.target.value)} className="px-4 py-2 border rounded-lg">
            <option value="revenue">Revenue</option>
            <option value="usage">Usage</option>
            <option value="subscribers">Subscribers</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700">From</label>
          <input type="date" value={fromDate} onChange={(e) => setFromDate(e.target.value)} className="px-4 py-2 border rounded-lg" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700">To</label>
          <input type="date" value={toDate} onChange={(e) => setToDate(e.target.value)} className="px-4 py-2 border rounded-lg" />
        </div>
        <button onClick={() => {}} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
          Generate
        </button>
      </div>

      {isLoading && <div className="text-center py-8">Loading...</div>}

      {reportData.length > 0 && (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                {Object.keys(reportData[0]).map((k) => <th key={k} className="px-4 py-3 text-left text-sm capitalize">{k}</th>)}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {reportData.map((row, i) => (
                <tr key={i} className="hover:bg-gray-50">
                  {Object.values(row).map((v, j) => <td key={j} className="px-4 py-3 text-sm">{v}</td>)}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
