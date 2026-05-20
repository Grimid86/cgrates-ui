import { useMemo } from 'react'
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts'

export default function CDRChart({ cdrs }) {
  const data = useMemo(() => {
    if (!cdrs || cdrs.length === 0) return []
    
    // Group by date
    const grouped = {}
    cdrs.forEach((cdr) => {
      const date = cdr.started_at?.split('T')[0] || 'Unknown'
      if (!grouped[date]) {
        grouped[date] = { date, count: 0, cost: 0, duration: 0 }
      }
      grouped[date].count += 1
      grouped[date].cost += cdr.cost || 0
      grouped[date].duration += cdr.duration_seconds || 0
    })
    
    return Object.values(grouped).sort((a, b) => a.date.localeCompare(b.date))
  }, [cdrs])

  if (data.length === 0) {
    return <div className="text-center text-gray-500 py-8">No CDR data</div>
  }

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="date" tick={{ fontSize: 12 }} />
          <YAxis tick={{ fontSize: 12 }} />
          <Tooltip />
          <Bar dataKey="count" fill="#007bff" name="Calls" />
          <Bar dataKey="cost" fill="#28a745" name="Cost" />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
