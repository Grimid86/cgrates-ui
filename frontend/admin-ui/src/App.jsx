import { Routes, Route } from 'react-router-dom'
import Layout from './components/layout/Layout'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import TenantsPage from './pages/TenantsPage'
import UsersPage from './pages/UsersPage'
import RBACPage from './pages/RBACPage'
import BrandingPage from './pages/BrandingPage'
import LocalizationPage from './pages/LocalizationPage'
import MonitoringPage from './pages/MonitoringPage'
import AuditLogPage from './pages/AuditLogPage'
import ProfilePage from './pages/ProfilePage'
import ProtectedRoute from './components/common/ProtectedRoute'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<Layout />}>
        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/tenants" element={<TenantsPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/rbac" element={<RBACPage />} />
          <Route path="/branding" element={<BrandingPage />} />
          <Route path="/localization" element={<LocalizationPage />} />
          <Route path="/monitoring" element={<MonitoringPage />} />
          <Route path="/audit-log" element={<AuditLogPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
