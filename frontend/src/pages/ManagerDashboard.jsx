import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../context/AuthContext'

const TABS = ['Employees', 'Attendance', 'Reports']

export default function ManagerDashboard() {
  const { employee, logout } = useAuth()
  const navigate = useNavigate()
  const [tab, setTab] = useState('Employees')

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="page">
      <header className="dashboard-header">
        <div>
          <h1>Manager Dashboard</h1>
          <p className="subtitle">
            {employee?.name} ({employee?.role})
          </p>
        </div>
        <div className="header-actions">
          <Link className="link-button" to="/qr">
            QR code
          </Link>
          <button className="link-button" onClick={handleLogout}>
            Log out
          </button>
        </div>
      </header>

      <nav className="tabs">
        {TABS.map((t) => (
          <button key={t} className={t === tab ? 'tab active' : 'tab'} onClick={() => setTab(t)}>
            {t}
          </button>
        ))}
      </nav>

      {tab === 'Employees' && <EmployeesTab />}
      {tab === 'Attendance' && <AttendanceTab />}
      {tab === 'Reports' && <ReportsTab />}
    </div>
  )
}

function EmployeesTab() {
  const { logout } = useAuth()
  const [employees, setEmployees] = useState([])
  const [error, setError] = useState('')
  const [form, setForm] = useState({
    employee_code: '',
    name: '',
    role: 'staff',
    pin: '',
    pay_type: 'hourly',
    pay_rate: '',
    is_temp: true,
  })

  async function refresh() {
    try {
      setEmployees((await api.listEmployees()) || [])
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  async function handleCreate(e) {
    e.preventDefault()
    setError('')
    try {
      await api.createEmployee({ ...form, pay_rate: parseFloat(form.pay_rate) || 0 })
      setForm({ employee_code: '', name: '', role: 'staff', pin: '', pay_type: 'hourly', pay_rate: '', is_temp: true })
      await refresh()
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  async function handleResetDevice(id) {
    try {
      await api.resetDevice(id)
      await refresh()
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  return (
    <div>
      <div className="card">
        <h2>Register employee</h2>
        <form onSubmit={handleCreate} className="form-grid">
          <label>
            Employee ID
            <input required value={form.employee_code} onChange={(e) => setForm({ ...form, employee_code: e.target.value })} />
          </label>
          <label>
            Name
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </label>
          <label>
            PIN
            <input required inputMode="numeric" value={form.pin} onChange={(e) => setForm({ ...form, pin: e.target.value })} />
          </label>
          <label>
            Role
            <select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
              <option value="staff">Staff</option>
              <option value="manager">Manager</option>
              <option value="admin">Admin</option>
            </select>
          </label>
          <label>
            Pay type
            <select
              value={form.pay_type}
              onChange={(e) => setForm({ ...form, pay_type: e.target.value, is_temp: e.target.value === 'hourly' })}
            >
              <option value="hourly">Hourly (temp/event)</option>
              <option value="monthly">Monthly (regular)</option>
            </select>
          </label>
          <label>
            {form.pay_type === 'hourly' ? 'Hourly rate' : 'Monthly salary'}
            <input required type="number" step="0.01" value={form.pay_rate} onChange={(e) => setForm({ ...form, pay_rate: e.target.value })} />
          </label>
          <button type="submit">Add employee</button>
        </form>
        {error && <p className="error">{error}</p>}
      </div>

      <div className="card">
        <h2>All employees</h2>
        <table>
          <thead>
            <tr>
              <th>Code</th>
              <th>Name</th>
              <th>Role</th>
              <th>Pay</th>
              <th>Device</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {employees.map((emp) => (
              <tr key={emp.id}>
                <td>{emp.employee_code}</td>
                <td>{emp.name}</td>
                <td>{emp.role}</td>
                <td>{emp.pay_type === 'hourly' ? `$${emp.pay_rate}/hr` : `$${emp.pay_rate}/mo`}</td>
                <td>{emp.device_id ? 'registered' : '—'}</td>
                <td>
                  {emp.device_id && (
                    <button className="link-button" onClick={() => handleResetDevice(emp.id)}>
                      Reset device
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function AttendanceTab() {
  const { logout } = useAuth()
  const [events, setEvents] = useState([])
  const [error, setError] = useState('')
  const [editingId, setEditingId] = useState(null)
  const [editValue, setEditValue] = useState('')

  async function refresh() {
    try {
      setEvents((await api.listAttendance()) || [])
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  function startEdit(ev) {
    setEditingId(ev.id)
    setEditValue(toLocalInputValue(ev.timestamp))
  }

  async function saveEdit(id) {
    try {
      await api.editAttendanceEvent(id, new Date(editValue).toISOString())
      setEditingId(null)
      await refresh()
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  return (
    <div className="card">
      <h2>Attendance log</h2>
      {error && <p className="error">{error}</p>}
      <table>
        <thead>
          <tr>
            <th>Employee</th>
            <th>Type</th>
            <th>Timestamp</th>
            <th>Flags</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {events.map((ev) => (
            <tr key={ev.id}>
              <td>#{ev.employee_id}</td>
              <td>{ev.event_type === 'check_in' ? 'In' : 'Out'}</td>
              <td>
                {editingId === ev.id ? (
                  <input type="datetime-local" value={editValue} onChange={(e) => setEditValue(e.target.value)} />
                ) : (
                  new Date(ev.timestamp).toLocaleString()
                )}
              </td>
              <td>{ev.auto_closed && <span className="tag">auto-closed</span>}</td>
              <td>
                {editingId === ev.id ? (
                  <button className="link-button" onClick={() => saveEdit(ev.id)}>
                    Save
                  </button>
                ) : (
                  <button className="link-button" onClick={() => startEdit(ev)}>
                    Edit
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function toLocalInputValue(isoString) {
  const d = new Date(isoString)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function ReportsTab() {
  const { logout } = useAuth()
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [rows, setRows] = useState([])
  const [error, setError] = useState('')

  async function refresh() {
    try {
      setRows((await api.monthlySummary(year, month)) || [])
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  useEffect(() => {
    refresh()
  }, [year, month])

  return (
    <div className="card">
      <h2>Monthly summary</h2>
      <div className="form-inline">
        <label>
          Year
          <input type="number" value={year} onChange={(e) => setYear(parseInt(e.target.value, 10))} />
        </label>
        <label>
          Month
          <input type="number" min="1" max="12" value={month} onChange={(e) => setMonth(parseInt(e.target.value, 10))} />
        </label>
      </div>
      {error && <p className="error">{error}</p>}
      <table>
        <thead>
          <tr>
            <th>Employee</th>
            <th>Pay type</th>
            <th>Hours</th>
            <th>Pay due</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.employee_id}>
              <td>{r.name}</td>
              <td>{r.pay_type}</td>
              <td>{r.total_hours.toFixed(2)}</td>
              <td>{r.pay_due != null ? `$${r.pay_due.toFixed(2)}` : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
