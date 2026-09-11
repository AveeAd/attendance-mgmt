import { Navigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function RequireAuth({ managerOnly = false, children }) {
  const { employee, isManager } = useAuth()

  if (!employee) return <Navigate to="/login" replace />
  if (managerOnly && !isManager) return <Navigate to="/" replace />

  return children
}
