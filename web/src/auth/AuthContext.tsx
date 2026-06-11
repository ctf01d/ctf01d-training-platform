import { createContext, useContext, useState, useCallback, useEffect, type ReactNode } from 'react'
import { getToken, setToken, clearToken } from '../api/auth'
import * as usersApi from '../api/users'
import type { User } from '../api/users'

interface AuthContextValue {
  user: User | null
  loading: boolean
  login: (userName: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
  isAdmin: boolean
  isPlayer: boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const refreshUser = useCallback(async () => {
    const token = getToken()
    if (!token) {
      setUser(null)
      setLoading(false)
      return
    }
    try {
      const { data } = await usersApi.getProfile()
      if (data) {
        setUser(data)
      } else {
        clearToken()
        setUser(null)
      }
    } catch {
      clearToken()
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refreshUser()
  }, [refreshUser])

  const login = useCallback(async (userName: string, password: string) => {
    const { data, error } = await usersApi.login({ user_name: userName, password })
    if (error || !data) {
      throw new Error(error?.message ?? 'Login failed')
    }
    setToken(data.token)
    setUser(data.user)
  }, [])

  const logout = useCallback(async () => {
    await usersApi.logout()
    clearToken()
    setUser(null)
  }, [])

  const isAdmin = user?.role === 'admin'
  const isPlayer = user?.role === 'player' || user?.role === 'admin'

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, refreshUser, isAdmin, isPlayer }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
