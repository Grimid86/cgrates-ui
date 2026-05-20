import { useMemo } from 'react'
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts'

const COLORS = ['#007bff', '#28a745', '#ffc107', '#dc3545']

export default function BalanceChart({ balances }) {
  const data = useMemo(() => {
    if (!balances) return []
    return balances.map((b) => ({
      name: b.type.replace('*', ''),
      value: b.value,
    })).filter((d) => d.value > 0)
  }, [balances])

  if (data.length === 0) {
    return <div className="text-center text-gray-500 py-8">No balance data</div>
  }

  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={80}
            paddingAngle={5}
            dataKey="value"
          >
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip formatter={(value) => value.toFixed(2)} />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  )
}
