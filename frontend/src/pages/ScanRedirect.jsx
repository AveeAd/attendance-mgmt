import { useEffect } from 'react'

// Public page — no login required. The QR code on the entrance screen
// points here instead of straight to /login.
//
// Many phone camera/QR-scanner apps open scanned links in their own
// ephemeral in-app WebView instead of the real browser. That WebView's
// localStorage doesn't persist between scans, so the device_id the login
// flow relies on (see api/client.js) gets regenerated every time — which
// then collides with the device already bound to the employee's account
// and wrongly reports "registered to a different device". Handing off to
// a real browser keeps the same device_id across scans, the way a normal
// bookmark would.
export default function ScanRedirect() {
  const loginUrl = `${window.location.origin}/login`

  useEffect(() => {
    if (/Android/i.test(navigator.userAgent)) {
      // intent:// is honored by Android's system WebView handoff in most
      // in-app browsers, and forces the link open in Chrome specifically.
      const bare = loginUrl.replace(/^https?:\/\//, '')
      window.location.href = `intent://${bare}#Intent;scheme=http;package=com.android.chrome;end`
    } else {
      window.location.replace(loginUrl)
    }
  }, [loginUrl])

  return (
    <div className="page page-narrow">
      <h1>Opening Check-In…</h1>
      <p className="subtitle">If nothing happens in a few seconds, tap below.</p>
      <a className="link-button" href={loginUrl} target="_blank" rel="noopener noreferrer">
        Open Check-In
      </a>
      <p className="muted">
        Still getting "registered to a different device"? Tap your browser's ⋯ menu and choose "Open in Safari" or
        "Open in Chrome" instead of scanning again.
      </p>
    </div>
  )
}
