package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/auth"
	"us.ztechai.zsms/backend/internal/database"
	"us.ztechai.zsms/backend/internal/domain"
)

type PhoneRepository struct {
	db                *database.Client
	tokenPrefix       string
	pairingCodeExpMin int
}

func NewPhoneRepository(db *database.Client, tokenPrefix string, pairingCodeExpMin int) *PhoneRepository {
	if tokenPrefix == "" {
		tokenPrefix = "zsms_dev_"
	}
	if pairingCodeExpMin <= 0 {
		pairingCodeExpMin = 10
	}
	return &PhoneRepository{
		db:                db,
		tokenPrefix:       tokenPrefix,
		pairingCodeExpMin: pairingCodeExpMin,
	}
}

// CreatePairingSession generates a 6-digit code and ephemeral session valid for the configured expiration duration.
func (r *PhoneRepository) CreatePairingSession(ctx context.Context, userID uuid.UUID) (*domain.PairingSession, error) {
	// Generate random 6-digit numeric code (e.g. 100000 - 999999)
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random pairing code: %w", err)
	}
	pairingCode := fmt.Sprintf("%06d", n.Int64()+100000)

	// Generate random 32-byte session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}
	sessionToken := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().Add(time.Duration(r.pairingCodeExpMin) * time.Minute)

	query := `
		INSERT INTO pairing_sessions (user_id, pairing_code, session_token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, user_id, pairing_code, session_token, expires_at, created_at
	`

	session := &domain.PairingSession{}
	err = r.db.DB.QueryRowContext(ctx, query, userID, pairingCode, sessionToken, expiresAt).Scan(
		&session.ID,
		&session.UserID,
		&session.PairingCode,
		&session.SessionToken,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert pairing session: %w", err)
	}

	return session, nil
}

// ClaimPairingSession completes device pairing, creates/updates phone record, and generates device credential.
func (r *PhoneRepository) ClaimPairingSession(
	ctx context.Context,
	pairingCode, deviceName, identifier, phoneNum, carrier string,
) (*domain.Phone, string, error) {
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	// 1. Locate valid active pairing session
	var sessionID uuid.UUID
	var userID uuid.UUID
	var expiresAt time.Time
	var claimedAt *time.Time

	findQuery := `
		SELECT id, user_id, expires_at, claimed_at
		FROM pairing_sessions
		WHERE pairing_code = $1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, findQuery, pairingCode).Scan(&sessionID, &userID, &expiresAt, &claimedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.New("invalid pairing code")
		}
		return nil, "", err
	}

	if claimedAt != nil {
		return nil, "", errors.New("pairing code has already been claimed")
	}

	if time.Now().After(expiresAt) {
		return nil, "", errors.New("pairing code has expired")
	}

	// 2. Upsert Phone record for user and device identifier
	upsertPhoneQuery := `
		INSERT INTO phones (
			user_id, name, device_identifier, phone_number, sim_carrier,
			status, last_seen_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, 'online', NOW(), NOW(), NOW()
		)
		ON CONFLICT (user_id, device_identifier) DO UPDATE SET
			name = EXCLUDED.name,
			phone_number = EXCLUDED.phone_number,
			sim_carrier = EXCLUDED.sim_carrier,
			status = 'online',
			last_seen_at = NOW(),
			updated_at = NOW()
		RETURNING id, user_id, name, device_identifier, phone_number, sim_carrier,
		          sim_slot_count, default_sim_slot, battery_level, battery_charging,
		          network_type, signal_strength, app_version, os_version, status,
		          last_seen_at, created_at, updated_at
	`

	phone := &domain.Phone{}
	err = tx.QueryRowContext(ctx, upsertPhoneQuery, userID, deviceName, identifier, phoneNum, carrier).Scan(
		&phone.ID,
		&phone.UserID,
		&phone.Name,
		&phone.DeviceIdentifier,
		&phone.PhoneNumber,
		&phone.SimCarrier,
		&phone.SimSlotCount,
		&phone.DefaultSimSlot,
		&phone.BatteryLevel,
		&phone.BatteryCharging,
		&phone.NetworkType,
		&phone.SignalStrength,
		&phone.AppVersion,
		&phone.OSVersion,
		&phone.Status,
		&phone.LastSeenAt,
		&phone.CreatedAt,
		&phone.UpdatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to upsert phone: %w", err)
	}

	// 3. Mark session claimed
	_, err = tx.ExecContext(ctx, `
		UPDATE pairing_sessions
		SET claimed_at = NOW(), claimed_by_phone_id = $1
		WHERE id = $2
	`, phone.ID, sessionID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to mark pairing session claimed: %w", err)
	}

	// 4. Generate persistent Device API Key
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", err
	}
	plaintextToken := r.tokenPrefix + hex.EncodeToString(tokenBytes)
	tokenHash := auth.HashToken(plaintextToken)
	prefixLen := len(r.tokenPrefix) + 8
	if prefixLen > 16 {
		prefixLen = 16
	}
	if prefixLen > len(plaintextToken) {
		prefixLen = len(plaintextToken)
	}
	tokenPrefix := plaintextToken[:prefixLen]

	// Revoke prior active credentials for this phone
	_, _ = tx.ExecContext(ctx, "UPDATE phone_credentials SET revoked_at = NOW() WHERE phone_id = $1 AND revoked_at IS NULL", phone.ID)

	// Insert new credential
	insertCredQuery := `
		INSERT INTO phone_credentials (phone_id, token_hash, token_prefix, created_at)
		VALUES ($1, $2, $3, NOW())
	`
	_, err = tx.ExecContext(ctx, insertCredQuery, phone.ID, tokenHash, tokenPrefix)
	if err != nil {
		return nil, "", fmt.Errorf("failed to insert phone credential: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}

	return phone, plaintextToken, nil
}

func (r *PhoneRepository) GetByID(ctx context.Context, phoneID, userID uuid.UUID) (*domain.Phone, error) {
	query := `
		SELECT id, user_id, name, device_identifier, phone_number, sim_carrier,
		       sim_slot_count, default_sim_slot, battery_level, battery_charging,
		       network_type, signal_strength, app_version, os_version, status,
		       last_seen_at, created_at, updated_at
		FROM phones
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`

	phone := &domain.Phone{}
	err := r.db.DB.QueryRowContext(ctx, query, phoneID, userID).Scan(
		&phone.ID,
		&phone.UserID,
		&phone.Name,
		&phone.DeviceIdentifier,
		&phone.PhoneNumber,
		&phone.SimCarrier,
		&phone.SimSlotCount,
		&phone.DefaultSimSlot,
		&phone.BatteryLevel,
		&phone.BatteryCharging,
		&phone.NetworkType,
		&phone.SignalStrength,
		&phone.AppVersion,
		&phone.OSVersion,
		&phone.Status,
		&phone.LastSeenAt,
		&phone.CreatedAt,
		&phone.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	r.computeDynamicStatus(phone)
	return phone, nil
}

func (r *PhoneRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Phone, error) {
	query := `
		SELECT id, user_id, name, device_identifier, phone_number, sim_carrier,
		       sim_slot_count, default_sim_slot, battery_level, battery_charging,
		       network_type, signal_strength, app_version, os_version, status,
		       last_seen_at, created_at, updated_at
		FROM phones
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var phones []*domain.Phone
	for rows.Next() {
		p := &domain.Phone{}
		if err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.Name,
			&p.DeviceIdentifier,
			&p.PhoneNumber,
			&p.SimCarrier,
			&p.SimSlotCount,
			&p.DefaultSimSlot,
			&p.BatteryLevel,
			&p.BatteryCharging,
			&p.NetworkType,
			&p.SignalStrength,
			&p.AppVersion,
			&p.OSVersion,
			&p.Status,
			&p.LastSeenAt,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		r.computeDynamicStatus(p)
		phones = append(phones, p)
	}

	return phones, nil
}

func (r *PhoneRepository) UpdateHeartbeat(
	ctx context.Context,
	phoneID uuid.UUID,
	battery int,
	charging bool,
	netType string,
	signal *int,
	appVer, osVer string,
) error {
	query := `
		UPDATE phones SET
			battery_level = $1,
			battery_charging = $2,
			network_type = $3,
			signal_strength = COALESCE($4, signal_strength),
			app_version = CASE WHEN $5 <> '' THEN $5 ELSE app_version END,
			os_version = CASE WHEN $6 <> '' THEN $6 ELSE os_version END,
			status = 'online',
			last_seen_at = NOW(),
			updated_at = NOW()
		WHERE id = $7 AND deleted_at IS NULL
	`
	_, err := r.db.DB.ExecContext(ctx, query, battery, charging, netType, signal, appVer, osVer, phoneID)
	return err
}

// computeDynamicStatus marks phone offline if last_seen_at > 120 seconds ago.
func (r *PhoneRepository) computeDynamicStatus(p *domain.Phone) {
	if p.Status == "disabled" {
		return
	}
	if p.LastSeenAt == nil || time.Since(*p.LastSeenAt) > 120*time.Second {
		p.Status = "offline"
	} else {
		p.Status = "online"
	}
}
