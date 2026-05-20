import { Routes, Route } from 'react-router-dom'
import Layout from './components/layout/Layout'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import BalancePage from './pages/BalancePage'
import CDRPage from './pages/CDRPage'
import ProfilePage from './pages/ProfilePage'
import TopUpPage from './pages/TopUpPage'
import ProtectedRoute from './components/common/ProtectedRoute'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<Layout />}>
        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/balance" element={<BalancePage />} />
          <Route path="/cdr" element={<CDRPage />} />
          <Route path="/profile" element={<ProfilePage />} />
          <Route path="/topup" element={<TopUpPage />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
