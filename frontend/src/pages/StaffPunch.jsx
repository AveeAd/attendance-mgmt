import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import ChangePinForm from '../components/ChangePinForm'
import { useAuth } from '../context/AuthContext'

const MIN_SHIFT_MS = 60 * 60 * 1000 // mirrors backend minShiftDuration

export default function StaffPunch() {
  const { employee, logout } = useAuth()
  const navigate = useNavigate()
  const [events, setEvents] = useState([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [now, setNow] = useState(Date.now())
  const [showChangePin, setShowChangePin] = useState(false)

  async function refresh() {
    try {
      const data = await api.myAttendance()
      setEvents(data || [])
    } catch (err) {
      if (err.isSessionExpired) return logout()
      setError(err.message)
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  const lastEvent = events[0]
  const nextAction = lastEvent?.event_type === 'check_in' ? 'check_out' : 'check_in'
  const eligibleAt = nextAction === 'check_out' ? new Date(lastEvent.timestamp).getTime() + MIN_SHIFT_MS : null
  const remainingMs = eligibleAt ? eligibleAt - now : 0
  const waitingToCheckOut = remainingMs > 0

  // Tick every second while waiting out the minimum-shift lock, so the
  // countdown and button re-enable without the user refreshing.
  useEffect(() => {
    if (!waitingToCheckOut) return
    const timer = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(timer)
  }, [waitingToCheckOut])

  async function handlePunch() {
    setBusy(true)
    setError('')
    try {
      await api.punch()
      await refresh()
    } catch (err) {
      if (err.isSessionExpired) return logout()
      if (err.remaining_seconds != null) {
        setError(`Not yet — you can check out in ${formatCountdown(err.remaining_seconds * 1000)}.`)
      } else {
        setError(err.message)
      }
      await refresh()
    } finally {
      setBusy(false)
    }
  }

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="page page-narrow">
      <h1>Hi, {employee?.name}</h1>
      <p className="subtitle">{employee?.employee_code}</p>

      <button className="punch-button" onClick={handlePunch} disabled={busy || waitingToCheckOut}>
        {busy ? 'Recording...' : nextAction === 'check_in' ? 'Check In' : 'Check Out'}
      </button>

      {waitingToCheckOut && (
        <p className="muted">
          You checked in {formatMinutesAgo(lastEvent.timestamp, now)} — check-out unlocks in {formatCountdown(remainingMs)}
          , to avoid accidental double-taps.
        </p>
      )}

      {error && <p className="error">{error}</p>}

      <h2>Today's activity</h2>
      <ul className="event-list">
        {events.slice(0, 10).map((e) => (
          <li key={e.id}>
            <span className={`badge ${e.event_type}`}>{e.event_type === 'check_in' ? 'In' : 'Out'}</span>
            {new Date(e.timestamp).toLocaleString()}
            {e.auto_closed && <span className="tag">auto-closed</span>}
          </li>
        ))}
        {events.length === 0 && <li className="muted">No events yet</li>}
      </ul>

      {showChangePin ? (
        <ChangePinForm onDone={() => setShowChangePin(false)} />
      ) : (
        <div className="footer-actions">
          <button className="link-button" onClick={() => setShowChangePin(true)}>
            Change PIN
          </button>
          <button className="link-button" onClick={handleLogout}>
            Log out
          </button>
        </div>
      )}
    </div>
  )
}

function formatCountdown(ms) {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${String(seconds).padStart(2, '0')}`
}

function formatMinutesAgo(timestamp, now) {
  const minutes = Math.max(0, Math.round((now - new Date(timestamp).getTime()) / 60000))
  return minutes <= 1 ? 'just now' : `${minutes} min ago`
}
