import { createContext, useContext, useState } from 'react'
import { api } from '../api/client'
import { clearSession, getStoredEmployee, getToken, storeSession } from '../api/client'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [employee, setEmployee] = useState(getStoredEmployee())
  const [token, setToken] = useState(getToken())

  async function login(employeeCode, pin) {
    const res = await api.login(employeeCode, pin)
    storeSession(res.token, res.employee)
    setToken(res.token)
    setEmployee(res.employee)
    return res.employee
  }

  async function logout() {
    try {
      await api.logout()
    } catch {
      // best-effort; clear local state regardless
    }
    clearSession()
    setToken(null)
    setEmployee(null)
  }

  const isManager = employee?.role === 'manager' || employee?.role === 'admin'

  return (
    <AuthContext.Provider value={{ employee, token, isManager, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
