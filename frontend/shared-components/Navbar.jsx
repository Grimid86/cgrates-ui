import { Link, useLocation } from 'react-router-dom'

export default function Navbar({ items = [], brand, onLogout }) {
  const location = useLocation()

  return (
    <nav className="bg-white border-b shadow-sm">
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center gap-8">
            {brand && (
              <Link to="/" className="text-xl font-bold text-gray-900">
                {brand}
              </Link>
            )}
            <div className="flex gap-4">
              {items.map((item) => (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`text-sm font-medium px-3 py-2 rounded transition ${
                    location.pathname === item.path
                      ? 'bg-gray-100 text-gray-900'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`}
                >
                  {item.label}
                </Link>
              ))}
            </div>
          </div>
          {onLogout && (
            <button
              onClick={onLogout}
              className="text-sm text-red-600 hover:text-red-700 font-medium"
            >
              Logout
            </button>
          )}
        </div>
      </div>
    </nav>
  )
}
