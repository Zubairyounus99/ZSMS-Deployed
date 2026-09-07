package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a ZSMS registered user.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"full_name"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"-"`
}

// Phone represents a registered Android gateway smartphone.
type Phone struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	Name             string     `json:"name"`
	DeviceIdentifier string     `json:"device_identifier"`
	PhoneNumber      string     `json:"phone_number"`
	SimCarrier       string     `json:"sim_carrier"`
	SimSlotCount     int        `json:"sim_slot_count"`
	DefaultSimSlot   int        `json:"default_sim_slot"`
	BatteryLevel     int        `json:"battery_level"`
	BatteryCharging  bool       `json:"battery_charging"`
	NetworkType      string     `json:"network_type"`
	SignalStrength   *int       `json:"signal_strength"`
	AppVersion       string     `json:"app_version"`
	OSVersion        string     `json:"os_version"`
	Status           string     `json:"status"` // "online", "offline", "pending", "disabled"
	LastSeenAt       *time.Time `json:"last_seen_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// PairingSession represents an ephemeral single-use session for linking an Android device.
type PairingSession struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	PairingCode       string     `json:"pairing_code"`
	SessionToken      string     `json:"session_token"`
	ExpiresAt         time.Time  `json:"expires_at"`
	ClaimedAt         *time.Time `json:"claimed_at"`
	ClaimedByPhoneID  *uuid.UUID `json:"claimed_by_phone_id"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Message represents an SMS message (inbound or outbound).
type Message struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	PhoneID      uuid.UUID  `json:"phone_id"`
	ThreadID     *uuid.UUID `json:"thread_id"`
	RequestID    *string    `json:"request_id"`
	Direction    string     `json:"direction"` // "outbound" or "inbound"
	Sender       string     `json:"sender"`
	Recipient    string     `json:"recipient"`
	Content      string     `json:"content"`
	Status       string     `json:"status"` // "queued", "sending", "sent", "delivered", "failed"
	ErrorCode    *string    `json:"error_code"`
	ErrorMessage *string    `json:"error_message"`
	SimSlot      int        `json:"sim_slot"`
	RetryCount   int        `json:"retry_count"`
	CreatedAt    time.Time  `json:"created_at"`
	SentAt       *time.Time `json:"sent_at"`
	DeliveredAt  *time.Time `json:"delivered_at"`
	FailedAt     *time.Time `json:"failed_at"`
	Events       []MessageEvent `json:"events,omitempty"`
}

// MessageThread represents a conversational grouping between a phone and a recipient.
type MessageThread struct {
	ID              uuid.UUID `json:"id"`
	PhoneID         uuid.UUID `json:"phone_id"`
	UserID          uuid.UUID `json:"user_id"`
	RecipientNumber string    `json:"recipient_number"`
	LastMessageAt   time.Time `json:"last_message_at"`
	Snippet         *string   `json:"snippet"`
	UnreadCount     int       `json:"unread_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MessageEvent represents an immutable audit log entry in a message's lifecycle.
type MessageEvent struct {
	ID           int64     `json:"id"`
	MessageID    uuid.UUID `json:"message_id"`
	Status       string    `json:"status"`
	Source       string    `json:"source"` // "api", "worker", "android", "carrier"
	ErrorMessage *string   `json:"error_message"`
	Metadata     string    `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
}
