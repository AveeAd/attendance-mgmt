import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import ChangePinForm from '../components/ChangePinForm'
import { useAuth } from '../context/AuthContext'

const TABS = ['Employees', 'Attendance', 'Reports']

export default function ManagerDashboard() {
  const { employee, logout } = useAuth()
  const navigate = useNavigate()
  const [tab, setTab] = useState('Employees')
  const [showChangePin, setShowChangePin] = useState(false)

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
          <button className="link-button" onClick={() => setShowChangePin((v) => !v)}>
            Change PIN
          </button>
          <button className="link-button" onClick={handleLogout}>
            Log out
          </button>
        </div>
      </header>

      {showChangePin && <ChangePinForm onDone={() => setShowChangePin(false)} />}

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
  const [resettingPinId, setResettingPinId] = useState(null)
  const [newPinValue, setNewPinValue] = useState('')
  const [confirmingDeleteId, setConfirmingDeleteId] = useState(null)
  const [showArchived, setShowArchived] = useState(false)
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

  async function handleResetPin(id) {
    if (!newPinValue) return
    try {
      await api.resetPin(id, newPinValue)
      setResettingPinId(null)
      setNewPinValue('')
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  async function handleArchive(id) {
    try {
      await api.archiveEmployee(id)
      await refresh()
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  async function handleActivate(id) {
    try {
      await api.activateEmployee(id)
      await refresh()
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  async function handleDelete(id) {
    try {
      await api.deleteEmployee(id)
      setConfirmingDeleteId(null)
      await refresh()
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setConfirmingDeleteId(null)
      setError(err.message)
    }
  }

  const visibleEmployees = showArchived ? employees : employees.filter((e) => e.is_active)

  return (
    <div>
      <div className="card">
        <h2>Register employee</h2>
        <form onSubmit={handleCreate} className="form-grid">
          <label>
            Employee ID
            <input
              required
              value={form.employee_code}
              onChange={(e) => setForm({ ...form, employee_code: e.target.value.toUpperCase() })}
            />
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
        <div className="section-header">
          <h2>All employees</h2>
          <label className="inline-checkbox">
            <input type="checkbox" checked={showArchived} onChange={(e) => setShowArchived(e.target.checked)} />
            Show archived
          </label>
        </div>
        <table>
          <thead>
            <tr>
              <th>Code</th>
              <th>Name</th>
              <th>Role</th>
              <th>Pay</th>
              <th>Status</th>
              <th>Device</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {visibleEmployees.map((emp) => (
              <tr key={emp.id} className={emp.is_active ? '' : 'archived-row'}>
                <td>{emp.employee_code}</td>
                <td>{emp.name}</td>
                <td>{emp.role}</td>
                <td>{emp.pay_type === 'hourly' ? `NPR ${emp.pay_rate}/hr` : `NPR ${emp.pay_rate}/mo`}</td>
                <td>{emp.is_active ? 'Active' : <span className="tag">Archived</span>}</td>
                <td>{emp.device_id ? 'registered' : '—'}</td>
                <td>
                  <div className="row-actions">
                    {emp.device_id && (
                      <button className="link-button" onClick={() => handleResetDevice(emp.id)}>
                        Reset device
                      </button>
                    )}
                    {resettingPinId === emp.id ? (
                      <>
                        <input
                          autoFocus
                          className="inline-input"
                          inputMode="numeric"
                          placeholder="New PIN"
                          value={newPinValue}
                          onChange={(e) => setNewPinValue(e.target.value)}
                        />
                        <button className="link-button" onClick={() => handleResetPin(emp.id)}>
                          Save
                        </button>
                        <button
                          className="link-button"
                          onClick={() => {
                            setResettingPinId(null)
                            setNewPinValue('')
                          }}
                        >
                          Cancel
                        </button>
                      </>
                    ) : (
                      <button className="link-button" onClick={() => setResettingPinId(emp.id)}>
                        Reset PIN
                      </button>
                    )}
                    {emp.is_active ? (
                      <button className="link-button" onClick={() => handleArchive(emp.id)}>
                        Archive
                      </button>
                    ) : (
                      <button className="link-button" onClick={() => handleActivate(emp.id)}>
                        Reactivate
                      </button>
                    )}
                    {confirmingDeleteId === emp.id ? (
                      <>
                        <span className="muted">Delete permanently?</span>
                        <button className="link-button danger" onClick={() => handleDelete(emp.id)}>
                          Confirm
                        </button>
                        <button className="link-button" onClick={() => setConfirmingDeleteId(null)}>
                          Cancel
                        </button>
                      </>
                    ) : (
                      <button className="link-button danger" onClick={() => setConfirmingDeleteId(emp.id)}>
                        Delete
                      </button>
                    )}
                  </div>
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
              <td>{r.pay_due != null ? `NPR ${r.pay_due.toFixed(2)}` : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
