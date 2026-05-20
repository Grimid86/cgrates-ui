import { Link, useLocation } from 'react-router-dom'
import { useState } from 'react'

export default function Sidebar({ items = [], brand, onLogout }) {
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)

  return (
    <div className={`flex flex-col h-screen bg-gray-900 text-white transition-all ${collapsed ? 'w-16' : 'w-64'}`}>
      {/* Brand */}
      <div className="h-16 flex items-center justify-between px-4 border-b border-gray-700">
        {!collapsed && (
          <Link to="/" className="text-lg font-bold truncate">
            {brand}
          </Link>
        )}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="p-1 rounded hover:bg-gray-700 text-gray-400 hover:text-white"
          title={collapsed ? 'Expand' : 'Collapse'}
        >
          {collapsed ? '→' : '←'}
        </button>
      </div>

      {/* Nav Items */}
      <nav className="flex-1 py-4 overflow-y-auto">
        {items.map((item) => {
          const active = location.pathname === item.path || location.pathname.startsWith(item.path + '/')
          return (
            <Link
              key={item.path}
              to={item.path}
              className={`flex items-center gap-3 px-4 py-3 mx-2 rounded-lg transition ${
                active
                  ? 'bg-gray-800 text-white font-medium'
                  : 'text-gray-400 hover:bg-gray-800 hover:text-white'
              }`}
              title={collapsed ? item.label : ''}
            >
              <span className="text-lg w-5 text-center">{item.icon || '•'}</span>
              {!collapsed && <span className="text-sm truncate">{item.label}</span>}
            </Link>
          )
        })}
      </nav>

      {/* Footer */}
      <div className="p-4 border-t border-gray-700">
        {onLogout && (
          <button
            onClick={onLogout}
            className={`flex items-center gap-3 text-red-400 hover:text-red-300 transition ${collapsed ? 'justify-center' : ''}`}
            title={collapsed ? 'Logout' : ''}
          >
            <span className="text-lg">⎋</span>
            {!collapsed && <span className="text-sm">Logout</span>}
          </button>
        )}
      </div>
    </div>
  )
}
