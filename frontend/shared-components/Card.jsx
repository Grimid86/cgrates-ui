export default function Card({ title, value, color = 'primary', children, className = '' }) {
  const colors = {
    primary: 'border-blue-500 text-blue-700',
    success: 'border-green-500 text-green-700',
    warning: 'border-yellow-500 text-yellow-700',
    danger: 'border-red-500 text-red-700',
    gray: 'border-gray-500 text-gray-700',
  }
  const borderColor = colors[color] || colors.primary

  return (
    <div className={`bg-white rounded-lg shadow p-5 border-l-4 ${borderColor} ${className}`}>
      {title && <p className="text-sm text-gray-500 mb-1">{title}</p>}
      {value !== undefined && <p className="text-2xl font-bold">{value}</p>}
      {children}
    </div>
  )
}
