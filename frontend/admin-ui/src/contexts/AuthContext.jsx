import { createContext, useContext, useState, useCallback } from 'react'
import api from '../services/api'
const AuthContext = createContext(null)
export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    const stored = localStorage.getItem('admin_user')
    return stored ? JSON.parse(stored) : null
  })
  const [isLoading, setIsLoading] = useState(false)
  const login = useCallback(async (email, password, mfaCode) => {
    setIsLoading(true)
    try {
      const { data } = await api.post('/auth/login', { email, password, mfa_code: mfaCode })
      localStorage.setItem('admin_access_token', data.access_token)
      localStorage.setItem('admin_refresh_token', data.refresh_token)
      localStorage.setItem('admin_user', JSON.stringify(data.user))
      setUser(data.user)
      return data
    } finally {
      setIsLoading(false)
    }
  }, [])
  const logout = useCallback(async () => {
    setIsLoading(true)
    try {
      await api.post('/auth/logout')
    } finally {
      localStorage.removeItem('admin_access_token')
      localStorage.removeItem('admin_refresh_token')
      localStorage.removeItem('admin_user')
      setUser(null)
      setIsLoading(false)
    }
  }, [])
  return <AuthContext.Provider value={{ user, login, logout, isAuthenticated: !!user, isLoading }}>{children}</AuthContext.Provider>
}
export const useAuth = () => useContext(AuthContext)
