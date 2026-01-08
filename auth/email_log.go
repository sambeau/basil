package auth

import (
	"context"
	"fmt"
	"time"
)

// EmailLog represents an email log entry
type EmailLog struct {
	ID                string
	UserID            *string
	Recipient         string
	EmailType         string // "verification", "recovery", "notification"
	Provider          string // "mailgun", "resend"
	ProviderMessageID *string
	Status            string // "sent", "failed"
	Error             *string
	CreatedAt         time.Time
}

// LogEmail logs an email send attempt to the database
func (d *DB) LogEmail(ctx context.Context, log *EmailLog) error {
	if log.ID == "" {
		log.ID = generateID("eml_")
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO email_logs (id, user_id, recipient, email_type, provider, provider_message_id, status, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.ExecContext(ctx, query,
		log.ID,
		log.UserID,
		log.Recipient,
		log.EmailType,
		log.Provider,
		log.ProviderMessageID,
		log.Status,
		log.Error,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("logging email: %w", err)
	}

	return nil
}

// GetEmailLogs retrieves email logs with optional filtering
func (d *DB) GetEmailLogs(ctx context.Context, userID *string, limit int) ([]EmailLog, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `
			SELECT id, user_id, recipient, email_type, provider, provider_message_id, status, error, created_at
			FROM email_logs
			WHERE user_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []interface{}{*userID, limit}
	} else {
		query = `
			SELECT id, user_id, recipient, email_type, provider, provider_message_id, status, error, created_at
			FROM email_logs
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []interface{}{limit}
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying email logs: %w", err)
	}
	defer rows.Close()

	var logs []EmailLog
	for rows.Next() {
		var log EmailLog
		var userID, providerMessageID, errorMsg *string

		err := rows.Scan(
			&log.ID,
			&userID,
			&log.Recipient,
			&log.EmailType,
			&log.Provider,
			&providerMessageID,
			&log.Status,
			&errorMsg,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning email log: %w", err)
		}

		log.UserID = userID
		log.ProviderMessageID = providerMessageID
		log.Error = errorMsg

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating email logs: %w", err)
	}

	return logs, nil
}
