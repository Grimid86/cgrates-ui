import { Link, useLocation } from 'react-router-dom'
import { useState } from 'react'
import { useI18n } from '../../contexts/I18nContext'

export default function Sidebar({ items = [], brand, onLogout }) {
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)
  const { locale, setLocale, t } = useI18n()

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
          const label = item.translationKey ? t(item.translationKey, 'common') : item.label
          return (
            <Link
              key={item.path}
              to={item.path}
              className={`flex items-center gap-3 px-4 py-3 mx-2 rounded-lg transition ${
                active
                  ? 'bg-gray-800 text-white font-medium'
                  : 'text-gray-400 hover:bg-gray-800 hover:text-white'
              }`}
              title={collapsed ? label : ''}
            >
              <span className="text-lg w-5 text-center">{item.icon || '•'}</span>
              {!collapsed && <span className="text-sm truncate">{label}</span>}
            </Link>
          )
        })}
      </nav>

      {/* Footer */}
      <div className="p-4 border-t border-gray-700 space-y-3">
        <div className={`${collapsed ? 'flex justify-center' : ''}`}>
          <select
            value={locale}
            onChange={(e) => setLocale(e.target.value)}
            className="bg-gray-800 text-white text-sm rounded px-2 py-1 border border-gray-600 focus:outline-none"
            title="Language"
          >
            <option value="en">EN</option>
            <option value="ru">RU</option>
            <option value="es">ES</option>
          </select>
        </div>
        {onLogout && (
          <button
            onClick={onLogout}
            className={`flex items-center gap-3 text-red-400 hover:text-red-300 transition ${collapsed ? 'justify-center' : ''}`}
            title={collapsed ? t('nav.logout', 'common') : ''}
          >
            <span className="text-lg">⎋</span>
            {!collapsed && <span className="text-sm">{t('nav.logout', 'common')}</span>}
          </button>
        )}
      </div>
    </div>
  )
}
