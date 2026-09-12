import QRCode from 'qrcode'
import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'

// Public page — no login required, so it can be displayed on an entrance
// screen/tablet or printed without anyone needing to sign in first.
export default function QrCodePage() {
  const canvasRef = useRef(null)
  const [addresses, setAddresses] = useState([])
  const [port, setPort] = useState('')
  const [url, setUrl] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    async function loadServerInfo() {
      try {
        const info = await api.serverInfo()
        setAddresses(info.addresses || [])
        setPort(info.port)
        if (info.addresses?.length) {
          setUrl(`http://${info.addresses[0]}:${info.port}/scan`)
        } else {
          // Fallback: can't detect a LAN address (e.g. offline dev machine).
          // The manager can still type the laptop's real address in by hand.
          setUrl(`${window.location.origin}/scan`)
        }
      } catch (err) {
        setError(err.message)
        setUrl(`${window.location.origin}/scan`)
      }
    }
    loadServerInfo()
  }, [])

  useEffect(() => {
    if (!url || !canvasRef.current) return
    QRCode.toCanvas(canvasRef.current, url, { width: 320, margin: 2 }, (err) => {
      if (err) setError(err.message)
    })
  }, [url])

  return (
    <div className="page page-narrow qr-page">
      <div className="no-print">
        <Link className="link-button" to="/manager">
          ← Back to dashboard
        </Link>
      </div>

      <h1>Scan to Check In</h1>
      <p className="subtitle">The Backyard — Staff Attendance</p>

      <canvas ref={canvasRef} className="qr-canvas" />

      <p className="qr-url">{url}</p>

      {error && <p className="error no-print">{error}</p>}

      <div className="no-print">
        {addresses.length > 1 && (
          <label>
            Laptop address
            <select value={url} onChange={(e) => setUrl(e.target.value)}>
              {addresses.map((addr) => (
                <option key={addr} value={`http://${addr}:${port}/scan`}>
                  {addr}
                </option>
              ))}
            </select>
          </label>
        )}

        <label>
          Or enter the URL manually
          <input value={url} onChange={(e) => setUrl(e.target.value)} />
        </label>

        <p className="muted">
          This should be the laptop's local network address, reachable from staff phones on the same office WiFi. Print
          this page and post it at the entrance — the laptop needs a fixed/reserved local IP so the code doesn't break
          on reboot.
        </p>

        <button onClick={() => window.print()}>Print</button>
      </div>
    </div>
  )
}
