import { Outlet } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import Sidebar from './Sidebar'

const navItems = [
  { path: '/', label: 'Dashboard', translationKey: 'nav.dashboard', icon: '📊' },
  { path: '/tenants', label: 'Tenants', translationKey: 'nav.tenants', icon: '🏢' },
  { path: '/users', label: 'Users', translationKey: 'nav.users', icon: '👥' },
  { path: '/rbac', label: 'RBAC', translationKey: 'nav.rbac', icon: '🔐' },
  { path: '/branding', label: 'Branding', translationKey: 'nav.branding', icon: '🎨' },
  { path: '/localization', label: 'Localization', translationKey: 'nav.localization', icon: '🌐' },
  { path: '/monitoring', label: 'Monitoring', translationKey: 'nav.monitoring', icon: '📡' },
  { path: '/audit-log', label: 'Audit Log', translationKey: 'nav.audit_log', icon: '📋' },
  { path: '/profile', label: 'Profile', translationKey: 'nav.profile', icon: '⚙️' },
]

export default function Layout() {
  const { logout } = useAuth()

  return (
    <div className="flex min-h-screen bg-gray-50">
      <Sidebar brand="Admin OSS" items={navItems} onLogout={logout} />
      <main className="flex-1 p-6 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
