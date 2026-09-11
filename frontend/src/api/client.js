const TOKEN_KEY = 'attendance.token'
const EMPLOYEE_KEY = 'attendance.employee'
const DEVICE_KEY = 'attendance.device_id'

// Each phone gets a stable random id on first use, persisted in
// localStorage. This is what the backend binds an account to.
export function getDeviceId() {
  let id = localStorage.getItem(DEVICE_KEY)
  if (!id) {
    id = generateId()
    localStorage.setItem(DEVICE_KEY, id)
  }
  return id
}

// crypto.randomUUID() only exists in "secure contexts" (HTTPS or
// localhost) — accessing the app over plain http:// from another device's
// LAN IP (the normal way staff phones reach it) is not a secure context,
// so that API is simply undefined there. Fall back to a manual v4-style
// random id, which doesn't need to be cryptographically strong here.
function generateId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function getStoredEmployee() {
  const raw = localStorage.getItem(EMPLOYEE_KEY)
  return raw ? JSON.parse(raw) : null
}

export function storeSession(token, employee) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(EMPLOYEE_KEY, JSON.stringify(employee))
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(EMPLOYEE_KEY)
}

async function request(path, { method = 'GET', body, auth = true } = {}) {
  const headers = {}
  if (body) headers['Content-Type'] = 'application/json'
  if (auth) {
    const token = getToken()
    if (token) headers['Authorization'] = `Bearer ${token}`
  }

  let res
  try {
    res = await fetch(`/api${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new Error("Can't reach the server. Check your WiFi connection and try again.")
  }

  // Only treat 401 as an expired session on authenticated requests — a
  // failed login attempt (bad PIN) also returns 401 but isn't that.
  if (auth && res.status === 401) {
    clearSession()
    const err = new Error('Your session has expired. Please log in again.')
    err.isSessionExpired = true
    throw err
  }

  if (!res.ok) {
    const contentType = res.headers.get('content-type') || ''
    if (contentType.includes('application/json')) {
      const data = await res.json()
      const err = new Error(data.error || `Request failed (${res.status})`)
      Object.assign(err, data)
      throw err
    }
    const text = await res.text()
    throw new Error(text || `Request failed (${res.status})`)
  }

  if (res.status === 204) return null
  return res.json()
}

export const api = {
  login: (employeeCode, pin) =>
    request('/login', { method: 'POST', auth: false, body: { employee_code: employeeCode, pin, device_id: getDeviceId() } }),
  logout: () => request('/logout', { method: 'POST' }),
  punch: () => request('/attendance/punch', { method: 'POST' }),
  myAttendance: () => request('/attendance/me'),
  changeMyPin: (currentPin, newPin) =>
    request('/me/change-pin', { method: 'POST', body: { current_pin: currentPin, new_pin: newPin } }),

  listEmployees: () => request('/employees'),
  createEmployee: (payload) => request('/employees', { method: 'POST', body: payload }),
  resetDevice: (employeeId) => request(`/employees/${employeeId}/reset-device`, { method: 'POST' }),
  resetPin: (employeeId, newPin) =>
    request(`/employees/${employeeId}/reset-pin`, { method: 'POST', body: { new_pin: newPin } }),
  archiveEmployee: (employeeId) => request(`/employees/${employeeId}/archive`, { method: 'POST' }),
  activateEmployee: (employeeId) => request(`/employees/${employeeId}/activate`, { method: 'POST' }),
  deleteEmployee: (employeeId) => request(`/employees/${employeeId}`, { method: 'DELETE' }),

  listAttendance: (params = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request(`/attendance${qs ? `?${qs}` : ''}`)
  },
  editAttendanceEvent: (id, timestamp) =>
    request(`/attendance/${id}`, { method: 'PATCH', body: { timestamp } }),

  monthlySummary: (year, month) => request(`/reports/monthly?year=${year}&month=${month}`),
  payrollExport: (year, month) => request(`/reports/payroll?year=${year}&month=${month}`),

  serverInfo: () => request('/server-info'),

  updateStatus: () => request('/update-status'),
  applyUpdate: () => request('/update/apply', { method: 'POST' }),
}
