import { createContext, useContext, useState, useCallback } from 'react'
import api from '../services/api'
const AuthContext = createContext(null)
export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    const stored = localStorage.getItem('admin_user')
    return stored ? JSON.parse(stored) : null
  })
  const login = useCallback(async (email, password, mfaCode) => {
    const { data } = await api.post('/auth/login', { email, password, mfa_code: mfaCode })
    localStorage.setItem('admin_access_token', data.access_token)
    localStorage.setItem('admin_refresh_token', data.refresh_token)
    localStorage.setItem('admin_user', JSON.stringify(data.user))
    setUser(data.user)
    return data
  }, [])
  const logout = useCallback(async () => {
    await api.post('/auth/logout')
    localStorage.removeItem('admin_access_token')
    localStorage.removeItem('admin_refresh_token')
    localStorage.removeItem('admin_user')
    setUser(null)
  }, [])
  return <AuthContext.Provider value={{ user, login, logout, isAuthenticated: !!user }}>{children}</AuthContext.Provider>
}
export const useAuth = () => useContext(AuthContext)
