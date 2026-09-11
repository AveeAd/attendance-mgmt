import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function Login() {
  const [employeeCode, setEmployeeCode] = useState('')
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const employee = await login(employeeCode.trim(), pin)
      const isManager = employee.role === 'manager' || employee.role === 'admin'
      navigate(isManager ? '/manager' : '/', { replace: true })
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="page page-narrow">
      <h1>The Backyard</h1>
      <p className="subtitle">Staff Attendance</p>
      <form onSubmit={handleSubmit} className="card">
        <label>
          Employee ID
          <input
            autoFocus
            value={employeeCode}
            onChange={(e) => setEmployeeCode(e.target.value.toUpperCase())}
            autoComplete="username"
          />
        </label>
        <label>
          PIN
          <input
            type="password"
            inputMode="numeric"
            value={pin}
            onChange={(e) => setPin(e.target.value)}
            autoComplete="current-password"
          />
        </label>
        {error && <p className="error">{error}</p>}
        <button type="submit" disabled={loading}>
          {loading ? 'Logging in...' : 'Log in'}
        </button>
      </form>
      <Link className="link-button" to="/qr">
        Show QR code
      </Link>
    </div>
  )
}
