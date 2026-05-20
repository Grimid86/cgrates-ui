export default function Card({ title, value, subtitle, icon, color = 'primary' }) {
  const colors = {
    primary: 'bg-blue-50 border-blue-200 text-blue-700',
    success: 'bg-green-50 border-green-200 text-green-700',
    warning: 'bg-yellow-50 border-yellow-200 text-yellow-700',
    danger: 'bg-red-50 border-red-200 text-red-700',
  }

  return (
    <div className={`rounded-lg border p-6 ${colors[color] || colors.primary}`}>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium opacity-80">{title}</p>
          <p className="text-2xl font-bold mt-1">{value}</p>
          {subtitle && <p className="text-xs mt-1 opacity-70">{subtitle}</p>}
        </div>
        {icon && <div className="text-3xl opacity-50">{icon}</div>}
      </div>
    </div>
  )
}
