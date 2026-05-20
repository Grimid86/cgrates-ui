export default function DataTable({ columns, data, loading }) {
  if (loading) return <div className="text-center py-8">Loading...</div>
  if (!data || data.length === 0) return <div className="text-center py-8 text-gray-500">No data</div>
  return (
    <div className="bg-white rounded-lg shadow overflow-hidden">
      <table className="w-full">
        <thead className="bg-gray-50"><tr>{columns.map(col => <th key={col.key} className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">{col.title}</th>)}</tr></thead>
        <tbody className="divide-y divide-gray-100">
          {data.map((row, i) => <tr key={i} className="hover:bg-gray-50">
            {columns.map(col => <td key={col.key} className="px-4 py-3 text-sm">{col.render ? col.render(row[col.key], row) : row[col.key]}</td>)}
          </tr>)}
        </tbody>
      </table>
    </div>
  )
}
