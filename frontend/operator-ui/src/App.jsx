import { Routes, Route } from 'react-router-dom'
import Layout from './components/layout/Layout'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import SubscribersPage from './pages/SubscribersPage'
import SubscriberDetailPage from './pages/SubscriberDetailPage'
import SubscriberNewPage from './pages/SubscriberNewPage'
import TariffsPage from './pages/TariffsPage'
import CDRPage from './pages/CDRPage'
import SessionsPage from './pages/SessionsPage'
import ReportsPage from './pages/ReportsPage'
import ProfilePage from './pages/ProfilePage'
import ProtectedRoute from './components/common/ProtectedRoute'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<Layout />}>
        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/subscribers" element={<SubscribersPage />} />
          <Route path="/subscribers/:id" element={<SubscriberDetailPage />} />
          <Route path="/subscribers/new" element={<SubscriberNewPage />} />
          <Route path="/tariffs" element={<TariffsPage />} />
          <Route path="/cdr" element={<CDRPage />} />
          <Route path="/sessions" element={<SessionsPage />} />
          <Route path="/reports" element={<ReportsPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
