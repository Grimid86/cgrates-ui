import { createContext, useContext, useState, useCallback } from 'react'
import api from '../services/api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    const stored = localStorage.getItem('selfcare_user')
    return stored ? JSON.parse(stored) : null
  })
  const [isLoading, setIsLoading] = useState(false)

  const login = useCallback(async (msisdn, pin) => {
    setIsLoading(true)
    try {
      const { data } = await api.post('/auth/login', { msisdn, pin })
      localStorage.setItem('selfcare_access_token', data.access_token)
      localStorage.setItem('selfcare_refresh_token', data.refresh_token)
      localStorage.setItem('selfcare_user', JSON.stringify(data.subscriber))
      setUser(data.subscriber)
      return data
    } finally {
      setIsLoading(false)
    }
  }, [])

  const logout = useCallback(async () => {
    await api.post('/auth/logout')
    localStorage.removeItem('selfcare_access_token')
    localStorage.removeItem('selfcare_refresh_token')
    localStorage.removeItem('selfcare_user')
    setUser(null)
  }, [])

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout, isAuthenticated: !!user }}>
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = () => useContext(AuthContext)
