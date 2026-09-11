import { useState } from 'react'
import { api } from '../api/client'

// Self-service PIN change, used from both the staff punch page and the
// manager dashboard. Requires the current PIN — for a forgotten PIN, a
// manager/admin resets it instead (see ManagerDashboard's "Reset PIN").
export default function ChangePinForm({ onDone }) {
  const [currentPin, setCurrentPin] = useState('')
  const [newPin, setNewPin] = useState('')
  const [confirmPin, setConfirmPin] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [busy, setBusy] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    if (newPin !== confirmPin) {
      setError('New PIN and confirmation do not match.')
      return
    }
    setBusy(true)
    try {
      await api.changeMyPin(currentPin, newPin)
      setSuccess(true)
      setCurrentPin('')
      setNewPin('')
      setConfirmPin('')
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  if (success) {
    return (
      <div className="card">
        <p>Your PIN has been updated.</p>
        <button onClick={onDone}>Done</button>
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="card">
      <label>
        Current PIN
        <input
          type="password"
          inputMode="numeric"
          value={currentPin}
          onChange={(e) => setCurrentPin(e.target.value)}
          autoComplete="current-password"
        />
      </label>
      <label>
        New PIN
        <input
          type="password"
          inputMode="numeric"
          value={newPin}
          onChange={(e) => setNewPin(e.target.value)}
          autoComplete="new-password"
        />
      </label>
      <label>
        Confirm new PIN
        <input
          type="password"
          inputMode="numeric"
          value={confirmPin}
          onChange={(e) => setConfirmPin(e.target.value)}
          autoComplete="new-password"
        />
      </label>
      {error && <p className="error">{error}</p>}
      <div className="form-inline">
        <button type="submit" disabled={busy}>
          {busy ? 'Saving...' : 'Change PIN'}
        </button>
        <button type="button" className="link-button" onClick={onDone}>
          Cancel
        </button>
      </div>
    </form>
  )
}
