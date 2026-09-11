package models

type Role string

const (
	RoleStaff   Role = "staff"
	RoleManager Role = "manager"
	RoleAdmin   Role = "admin"
)

type PayType string

const (
	PayTypeMonthly PayType = "monthly"
	PayTypeHourly  PayType = "hourly"
)

type EventType string

const (
	EventCheckIn  EventType = "check_in"
	EventCheckOut EventType = "check_out"
)

type Employee struct {
	ID           int64   `json:"id"`
	EmployeeCode string  `json:"employee_code"`
	Name         string  `json:"name"`
	Role         Role    `json:"role"`
	PinHash      string  `json:"-"`
	PayType      PayType `json:"pay_type"`
	PayRate      float64 `json:"pay_rate"`
	IsTemp       bool    `json:"is_temp"`
	DeviceID     *string `json:"device_id,omitempty"`
	IsActive     bool    `json:"is_active"`
	CreatedAt    string  `json:"created_at"`
}

type AttendanceEvent struct {
	ID         int64     `json:"id"`
	EmployeeID int64     `json:"employee_id"`
	EventType  EventType `json:"event_type"`
	Timestamp  string    `json:"timestamp"`
	AutoClosed bool      `json:"auto_closed"`
	CreatedAt  string    `json:"created_at"`
}

type AttendanceEditLog struct {
	ID                 int64  `json:"id"`
	AttendanceEventID  int64  `json:"attendance_event_id"`
	EditedBy           int64  `json:"edited_by"`
	EditedAt           string `json:"edited_at"`
	FieldChanged       string `json:"field_changed"`
	OldValue           string `json:"old_value"`
	NewValue           string `json:"new_value"`
}

type Session struct {
	Token      string `json:"token"`
	EmployeeID int64  `json:"employee_id"`
	DeviceID   string `json:"device_id"`
	CreatedAt  string `json:"created_at"`
	ExpiresAt  string `json:"expires_at"`
}
