import { Navigate, Route, Routes } from 'react-router-dom'
import RequireAuth from './components/RequireAuth'
import { useAuth } from './context/AuthContext'
import Login from './pages/Login'
import ManagerDashboard from './pages/ManagerDashboard'
import QrCode from './pages/QrCode'
import StaffPunch from './pages/StaffPunch'

export default function App() {
  const { isManager } = useAuth()

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <RequireAuth>
            {isManager ? <Navigate to="/manager" replace /> : <StaffPunch />}
          </RequireAuth>
        }
      />
      <Route
        path="/manager"
        element={
          <RequireAuth managerOnly>
            <ManagerDashboard />
          </RequireAuth>
        }
      />
      <Route path="/qr" element={<QrCode />} />
    </Routes>
  )
}
