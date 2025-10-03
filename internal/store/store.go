package store

import (
	"context"
	"database/sql"
)

type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store { return &Store{DB: db} }

func (s *Store) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) InsertCampaign(ctx context.Context, tx *sql.Tx, name, body string, scheduledAt any) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
	INSERT INTO campaigns (name,body,scheduled_at,status)
	VALUES ($1,$2,$3,'queued') RETURNING id`, name, body, scheduledAt).Scan(&id)
	return id, err
}

func (s *Store) InsertRecipient(ctx context.Context, tx *sql.Tx, campaignID int64, address string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO recipients (campaign_id, address)
		VALUES ($1,$2) RETURNING id
	`, campaignID, address).Scan(&id)
	return id, err
}
func (s *Store) GetCampaignBody(ctx context.Context, dbOrTx interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, campaignID int64) (string, error) {
	var body string
	err := dbOrTx.QueryRowContext(ctx, `SELECT body FROM campaigns WHERE id=$1`, campaignID).Scan(&body)
	return body, err
}

func (s *Store) InsertMessagePending(ctx context.Context, tx *sql.Tx, campaignID, recipientID int64) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO messages (campaign_id, recipient_id, status)
		VALUES ($1,$2,'pending')
	`, campaignID, recipientID)
	return err
}

func (s *Store) MarkMessageSent(ctx context.Context, msg *sql.DB, campaignID, recipientID int64) error {
	_, err := msg.ExecContext(ctx, `
		UPDATE messages
		   SET status='sent', sent_at=NOW(), last_error=NULL
		 WHERE campaign_id=$1 AND recipient_id=$2
	`, campaignID, recipientID)
	return err
}

func (s *Store) MarkMessageFailed(ctx context.Context, msg *sql.DB, campaignID, recipientID int64, lastErr string) error {
	_, err := msg.ExecContext(ctx, `
		UPDATE messages
		   SET status='failed', last_error=$1
		 WHERE campaign_id=$2 AND recipient_id=$3
	`, lastErr, campaignID, recipientID)
	return err
}
