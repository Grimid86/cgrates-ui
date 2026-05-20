import { Outlet } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import Sidebar from './Sidebar'

const navItems = [
  { path: '/', label: 'Dashboard', translationKey: 'nav.dashboard', icon: '📊' },
  { path: '/subscribers', label: 'Subscribers', translationKey: 'nav.subscribers', icon: '👤' },
  { path: '/tariffs', label: 'Tariffs', translationKey: 'nav.tariffs', icon: '📜' },
  { path: '/cdr', label: 'CDR', translationKey: 'nav.cdr', icon: '📞' },
  { path: '/sessions', label: 'Sessions', translationKey: 'nav.sessions', icon: '🔌' },
  { path: '/reports', label: 'Reports', translationKey: 'nav.reports', icon: '📈' },
  { path: '/profile', label: 'Profile', translationKey: 'nav.profile', icon: '⚙️' },
]

export default function Layout() {
  const { logout } = useAuth()

  return (
    <div className="flex min-h-screen bg-gray-50">
      <Sidebar brand="Operator BSS" items={navItems} onLogout={logout} />
      <main className="flex-1 p-6 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
