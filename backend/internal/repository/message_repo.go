package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/database"
	"us.ztechai.zsms/backend/internal/domain"
)

type MessageRepository struct {
	db *database.Client
}

func NewMessageRepository(db *database.Client) *MessageRepository {
	return &MessageRepository{db: db}
}

// FindOrCreateThread locates or inserts a message thread between a phone and recipient.
func (r *MessageRepository) FindOrCreateThread(
	ctx context.Context,
	phoneID, userID uuid.UUID,
	recipientNumber, snippet string,
) (*domain.MessageThread, error) {
	query := `
		INSERT INTO message_threads (phone_id, user_id, recipient_number, last_message_at, snippet, updated_at)
		VALUES ($1, $2, $3, NOW(), $4, NOW())
		ON CONFLICT (phone_id, recipient_number) DO UPDATE SET
			last_message_at = NOW(),
			snippet = EXCLUDED.snippet,
			updated_at = NOW()
		RETURNING id, phone_id, user_id, recipient_number, last_message_at, snippet, unread_count, created_at, updated_at
	`

	thread := &domain.MessageThread{}
	err := r.db.DB.QueryRowContext(ctx, query, phoneID, userID, recipientNumber, snippet).Scan(
		&thread.ID,
		&thread.PhoneID,
		&thread.UserID,
		&thread.RecipientNumber,
		&thread.LastMessageAt,
		&thread.Snippet,
		&thread.UnreadCount,
		&thread.CreatedAt,
		&thread.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create thread: %w", err)
	}

	return thread, nil
}

// CreateOutboundMessage inserts an outbound message and its initial queued event.
func (r *MessageRepository) CreateOutboundMessage(
	ctx context.Context,
	userID, phoneID uuid.UUID,
	senderPhone, recipientNumber, content string,
	simSlot int,
	requestID *string,
) (*domain.Message, error) {
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Check idempotency if request_id is provided
	if requestID != nil && *requestID != "" {
		existing := &domain.Message{}
		checkQuery := `
			SELECT id, user_id, phone_id, thread_id, request_id, direction, sender, recipient, content, status, sim_slot, created_at
			FROM messages
			WHERE request_id = $1
			LIMIT 1
		`
		err = tx.QueryRowContext(ctx, checkQuery, *requestID).Scan(
			&existing.ID,
			&existing.UserID,
			&existing.PhoneID,
			&existing.ThreadID,
			&existing.RequestID,
			&existing.Direction,
			&existing.Sender,
			&existing.Recipient,
			&existing.Content,
			&existing.Status,
			&existing.SimSlot,
			&existing.CreatedAt,
		)
		if err == nil {
			// Idempotent hit: return existing message without re-inserting
			return existing, nil
		}
	}

	// 2. Find or create conversational thread
	threadQuery := `
		INSERT INTO message_threads (phone_id, user_id, recipient_number, last_message_at, snippet, updated_at)
		VALUES ($1, $2, $3, NOW(), $4, NOW())
		ON CONFLICT (phone_id, recipient_number) DO UPDATE SET
			last_message_at = NOW(),
			snippet = EXCLUDED.snippet,
			updated_at = NOW()
		RETURNING id
	`
	var threadID uuid.UUID
	err = tx.QueryRowContext(ctx, threadQuery, phoneID, userID, recipientNumber, content).Scan(&threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to link thread: %w", err)
	}

	// 3. Insert outbound message row
	insertMsgQuery := `
		INSERT INTO messages (
			user_id, phone_id, thread_id, request_id, direction, sender, recipient, content,
			status, sim_slot, created_at
		) VALUES (
			$1, $2, $3, $4, 'outbound', $5, $6, $7, 'queued', $8, NOW()
		)
		RETURNING id, user_id, phone_id, thread_id, request_id, direction, sender, recipient, content,
		          status, sim_slot, retry_count, created_at
	`

	msg := &domain.Message{}
	err = tx.QueryRowContext(
		ctx, insertMsgQuery,
		userID, phoneID, threadID, requestID, senderPhone, recipientNumber, content, simSlot,
	).Scan(
		&msg.ID,
		&msg.UserID,
		&msg.PhoneID,
		&msg.ThreadID,
		&msg.RequestID,
		&msg.Direction,
		&msg.Sender,
		&msg.Recipient,
		&msg.Content,
		&msg.Status,
		&msg.SimSlot,
		&msg.RetryCount,
		&msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert outbound message: %w", err)
	}

	// 4. Insert initial message event
	insertEventQuery := `
		INSERT INTO message_events (message_id, status, source, created_at)
		VALUES ($1, 'queued', 'api', NOW())
	`
	_, err = tx.ExecContext(ctx, insertEventQuery, msg.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert initial message event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return msg, nil
}

// IngestInboundMessage records a message received by the physical SIM card.
func (r *MessageRepository) IngestInboundMessage(
	ctx context.Context,
	phoneID uuid.UUID,
	sender, recipient, content string,
	simSlot int,
	receivedAt time.Time,
) (*domain.Message, error) {
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Get user_id for this phone
	var userID uuid.UUID
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM phones WHERE id = $1", phoneID).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("phone not found: %w", err)
	}

	// 2. Deduplication check (same phone, sender, content within last 60 seconds)
	var existingID uuid.UUID
	dedupQuery := `
		SELECT id FROM messages
		WHERE phone_id = $1 AND direction = 'inbound' AND sender = $2 AND content = $3
		  AND created_at >= NOW() - INTERVAL '60 seconds'
		LIMIT 1
	`
	err = tx.QueryRowContext(ctx, dedupQuery, phoneID, sender, content).Scan(&existingID)
	if err == nil {
		// Already ingested: return early
		return r.GetByID(ctx, existingID, userID)
	}

	// 3. Find or create thread
	threadQuery := `
		INSERT INTO message_threads (phone_id, user_id, recipient_number, last_message_at, snippet, unread_count, updated_at)
		VALUES ($1, $2, $3, NOW(), $4, 1, NOW())
		ON CONFLICT (phone_id, recipient_number) DO UPDATE SET
			last_message_at = NOW(),
			snippet = EXCLUDED.snippet,
			unread_count = message_threads.unread_count + 1,
			updated_at = NOW()
		RETURNING id
	`
	var threadID uuid.UUID
	err = tx.QueryRowContext(ctx, threadQuery, phoneID, userID, sender, content).Scan(&threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to link thread for inbound message: %w", err)
	}

	// 4. Insert message as delivered
	insertMsgQuery := `
		INSERT INTO messages (
			user_id, phone_id, thread_id, direction, sender, recipient, content,
			status, sim_slot, created_at, delivered_at
		) VALUES (
			$1, $2, $3, 'inbound', $4, $5, $6, 'delivered', $7, $8, $8
		)
		RETURNING id, user_id, phone_id, thread_id, direction, sender, recipient, content,
		          status, sim_slot, retry_count, created_at, delivered_at
	`

	msg := &domain.Message{}
	err = tx.QueryRowContext(
		ctx, insertMsgQuery,
		userID, phoneID, threadID, sender, recipient, content, simSlot, receivedAt,
	).Scan(
		&msg.ID,
		&msg.UserID,
		&msg.PhoneID,
		&msg.ThreadID,
		&msg.Direction,
		&msg.Sender,
		&msg.Recipient,
		&msg.Content,
		&msg.Status,
		&msg.SimSlot,
		&msg.RetryCount,
		&msg.CreatedAt,
		&msg.DeliveredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert inbound message: %w", err)
	}

	// 5. Insert event
	_, err = tx.ExecContext(ctx, `
		INSERT INTO message_events (message_id, status, source, created_at)
		VALUES ($1, 'delivered', 'android', NOW())
	`, msg.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return msg, nil
}

// UpdateResult updates message state based on device transmission or delivery report.
func (r *MessageRepository) UpdateResult(
	ctx context.Context,
	messageID, phoneID uuid.UUID,
	status, errCode, errMsg string,
) error {
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var updateQuery string
	switch status {
	case "sent":
		updateQuery = `
			UPDATE messages SET
				status = 'sent',
				sent_at = COALESCE(sent_at, NOW()),
				error_code = NULL,
				error_message = NULL
			WHERE id = $1 AND phone_id = $2
		`
	case "delivered":
		updateQuery = `
			UPDATE messages SET
				status = 'delivered',
				delivered_at = NOW(),
				error_code = NULL,
				error_message = NULL
			WHERE id = $1 AND phone_id = $2
		`
	case "failed":
		updateQuery = `
			UPDATE messages SET
				status = 'failed',
				failed_at = NOW(),
				error_code = $3,
				error_message = $4
			WHERE id = $1 AND phone_id = $2
		`
	default:
		return fmt.Errorf("unsupported status update: %s", status)
	}

	if status == "failed" {
		_, err = tx.ExecContext(ctx, updateQuery, messageID, phoneID, errCode, errMsg)
	} else {
		_, err = tx.ExecContext(ctx, updateQuery, messageID, phoneID)
	}
	if err != nil {
		return fmt.Errorf("failed to update message result: %w", err)
	}

	// Insert event
	_, err = tx.ExecContext(ctx, `
		INSERT INTO message_events (message_id, status, source, error_message, created_at)
		VALUES ($1, $2, 'android', $3, NOW())
	`, messageID, status, errMsg)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *MessageRepository) GetByID(ctx context.Context, messageID, userID uuid.UUID) (*domain.Message, error) {
	query := `
		SELECT id, user_id, phone_id, thread_id, request_id, direction, sender, recipient, content,
		       status, error_code, error_message, sim_slot, retry_count, created_at, sent_at, delivered_at, failed_at
		FROM messages
		WHERE id = $1 AND user_id = $2
		LIMIT 1
	`

	msg := &domain.Message{}
	err := r.db.DB.QueryRowContext(ctx, query, messageID, userID).Scan(
		&msg.ID,
		&msg.UserID,
		&msg.PhoneID,
		&msg.ThreadID,
		&msg.RequestID,
		&msg.Direction,
		&msg.Sender,
		&msg.Recipient,
		&msg.Content,
		&msg.Status,
		&msg.ErrorCode,
		&msg.ErrorMessage,
		&msg.SimSlot,
		&msg.RetryCount,
		&msg.CreatedAt,
		&msg.SentAt,
		&msg.DeliveredAt,
		&msg.FailedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Fetch chronological event trail
	eventRows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, message_id, status, source, error_message, metadata::text, created_at
		FROM message_events
		WHERE message_id = $1
		ORDER BY created_at ASC
	`, messageID)
	if err == nil {
		defer eventRows.Close()
		for eventRows.Next() {
			ev := domain.MessageEvent{}
			if err := eventRows.Scan(&ev.ID, &ev.MessageID, &ev.Status, &ev.Source, &ev.ErrorMessage, &ev.Metadata, &ev.CreatedAt); err == nil {
				msg.Events = append(msg.Events, ev)
			}
		}
	}

	return msg, nil
}

func (r *MessageRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	phoneID *uuid.UUID,
	direction, status string,
	page, limit int,
) ([]*domain.Message, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int
	countQuery := "SELECT COUNT(*) FROM messages WHERE user_id = $1"
	args := []interface{}{userID}

	if phoneID != nil {
		args = append(args, *phoneID)
		countQuery += fmt.Sprintf(" AND phone_id = $%d", len(args))
	}
	if direction != "" {
		args = append(args, direction)
		countQuery += fmt.Sprintf(" AND direction = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		countQuery += fmt.Sprintf(" AND status = $%d", len(args))
	}

	if err := r.db.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectQuery := `
		SELECT id, user_id, phone_id, thread_id, request_id, direction, sender, recipient, content,
		       status, error_code, error_message, sim_slot, retry_count, created_at, sent_at, delivered_at, failed_at
		FROM messages
		WHERE user_id = $1
	`
	selectArgs := []interface{}{userID}

	if phoneID != nil {
		selectArgs = append(selectArgs, *phoneID)
		selectQuery += fmt.Sprintf(" AND phone_id = $%d", len(selectArgs))
	}
	if direction != "" {
		selectArgs = append(selectArgs, direction)
		selectQuery += fmt.Sprintf(" AND direction = $%d", len(selectArgs))
	}
	if status != "" {
		selectArgs = append(selectArgs, status)
		selectQuery += fmt.Sprintf(" AND status = $%d", len(selectArgs))
	}

	selectQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", limit, offset)

	rows, err := r.db.DB.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		if err := rows.Scan(
			&msg.ID,
			&msg.UserID,
			&msg.PhoneID,
			&msg.ThreadID,
			&msg.RequestID,
			&msg.Direction,
			&msg.Sender,
			&msg.Recipient,
			&msg.Content,
			&msg.Status,
			&msg.ErrorCode,
			&msg.ErrorMessage,
			&msg.SimSlot,
			&msg.RetryCount,
			&msg.CreatedAt,
			&msg.SentAt,
			&msg.DeliveredAt,
			&msg.FailedAt,
		); err != nil {
			return nil, 0, err
		}
		messages = append(messages, msg)
	}

	return messages, total, nil
}

func (r *MessageRepository) ListThreadsByUser(ctx context.Context, userID uuid.UUID, phoneID *uuid.UUID) ([]*domain.MessageThread, error) {
	query := `
		SELECT id, phone_id, user_id, recipient_number, last_message_at, snippet, unread_count, created_at, updated_at
		FROM message_threads
		WHERE user_id = $1
	`
	args := []interface{}{userID}

	if phoneID != nil {
		args = append(args, *phoneID)
		query += " AND phone_id = $2"
	}

	query += " ORDER BY last_message_at DESC LIMIT 100"

	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []*domain.MessageThread
	for rows.Next() {
		t := &domain.MessageThread{}
		if err := rows.Scan(
			&t.ID,
			&t.PhoneID,
			&t.UserID,
			&t.RecipientNumber,
			&t.LastMessageAt,
			&t.Snippet,
			&t.UnreadCount,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		threads = append(threads, t)
	}

	return threads, nil
}

func (r *MessageRepository) GetThreadMessages(ctx context.Context, threadID, userID uuid.UUID, page, limit int) ([]*domain.Message, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	query := `
		SELECT id, user_id, phone_id, thread_id, request_id, direction, sender, recipient, content,
		       status, error_code, error_message, sim_slot, retry_count, created_at, sent_at, delivered_at, failed_at
		FROM messages
		WHERE thread_id = $1 AND user_id = $2
		ORDER BY created_at ASC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.DB.QueryContext(ctx, query, threadID, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		if err := rows.Scan(
			&msg.ID,
			&msg.UserID,
			&msg.PhoneID,
			&msg.ThreadID,
			&msg.RequestID,
			&msg.Direction,
			&msg.Sender,
			&msg.Recipient,
			&msg.Content,
			&msg.Status,
			&msg.ErrorCode,
			&msg.ErrorMessage,
			&msg.SimSlot,
			&msg.RetryCount,
			&msg.CreatedAt,
			&msg.SentAt,
			&msg.DeliveredAt,
			&msg.FailedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Mark thread unread_count = 0
	_, _ = r.db.DB.ExecContext(ctx, "UPDATE message_threads SET unread_count = 0 WHERE id = $1", threadID)

	return messages, nil
}

// GetDashboardStats aggregates counts for the authenticated user without fake data.
func (r *MessageRepository) GetDashboardStats(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE direction = 'outbound' AND created_at >= CURRENT_DATE) AS sent_today,
			COUNT(*) FILTER (WHERE direction = 'inbound' AND created_at >= CURRENT_DATE) AS received_today,
			COUNT(*) FILTER (WHERE status = 'delivered') AS delivered_total,
			COUNT(*) FILTER (WHERE status = 'sent') AS sent_total,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed_total,
			COUNT(*) FILTER (WHERE status = 'queued') AS queued_total
		FROM messages
		WHERE user_id = $1
	`

	var sentToday, receivedToday, deliveredTotal, sentTotal, failedTotal, queuedTotal int
	err := r.db.DB.QueryRowContext(ctx, query, userID).Scan(
		&sentToday,
		&receivedToday,
		&deliveredTotal,
		&sentTotal,
		&failedTotal,
		&queuedTotal,
	)
	if err != nil {
		return nil, err
	}

	return map[string]int{
		"sent_today":      sentToday,
		"received_today":  receivedToday,
		"delivered_total": deliveredTotal,
		"sent_total":      sentTotal,
		"failed_total":    failedTotal,
		"queued_total":    queuedTotal,
	}, nil
}
