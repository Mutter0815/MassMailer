package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strconv"
	"strings"
	"time"
)

type Store struct {
	DB *sql.DB
}
type CampaignRow struct {
	ID          int64
	Name        string
	Body        string
	ScheduledAt time.Time
	Status      string
	CreatedAt   time.Time
}

type CampaignStats struct {
	Total   int
	Pending int
	Sent    int
	Failed  int
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

func (s *Store) GetCampaign(ctx context.Context, id int64) (CampaignRow, error) {
	var c CampaignRow
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, name, body, scheduled_at, status, created_at
		FROM campaigns
		WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Body, &c.ScheduledAt, &c.Status, &c.CreatedAt)
	if err != nil {
		return CampaignRow{}, err
	}
	return c, nil
}

func (s *Store) GetCampaignStats(ctx context.Context, id int64) (CampaignStats, error) {
	var st CampaignStats
	err := s.DB.QueryRowContext(ctx, `
		SELECT
		  COUNT(*)                                         AS total,
		  COUNT(*) FILTER (WHERE status='pending')         AS pending,
		  COUNT(*) FILTER (WHERE status='sent')            AS sent,
		  COUNT(*) FILTER (WHERE status='failed')          AS failed
		FROM messages
		WHERE campaign_id = $1
	`, id).Scan(&st.Total, &st.Pending, &st.Sent, &st.Failed)
	if err != nil {
		return CampaignStats{}, err
	}
	return st, nil
}

func (s *Store) ListCampaigns(ctx context.Context, limit, offset int) ([]CampaignRow, []CampaignStats, error) {
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, name, body, scheduled_at, status, created_at
		FROM campaigns
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var campaigns []CampaignRow
	var ids []int64
	for rows.Next() {
		var c CampaignRow
		if err := rows.Scan(&c.ID, &c.Name, &c.Body, &c.ScheduledAt, &c.Status, &c.CreatedAt); err != nil {
			return nil, nil, err
		}
		campaigns = append(campaigns, c)
		ids = append(ids, c.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(campaigns) == 0 {
		return campaigns, []CampaignStats{}, nil
	}

	statRows, err := s.DB.QueryContext(ctx, `
		SELECT campaign_id,
		       COUNT(*)                                         AS total,
		       COUNT(*) FILTER (WHERE status='pending')         AS pending,
		       COUNT(*) FILTER (WHERE status='sent')            AS sent,
		       COUNT(*) FILTER (WHERE status='failed')          AS failed
		FROM messages
		WHERE campaign_id = ANY($1)
		GROUP BY campaign_id
	`, int64Slice(ids))
	if err != nil {
		return nil, nil, err
	}
	defer statRows.Close()

	statsByID := make(map[int64]CampaignStats, len(ids))
	for statRows.Next() {
		var id int64
		var st CampaignStats
		if err := statRows.Scan(&id, &st.Total, &st.Pending, &st.Sent, &st.Failed); err != nil {
			return nil, nil, err
		}
		statsByID[id] = st
	}
	if err := statRows.Err(); err != nil {
		return nil, nil, err
	}

	out := make([]CampaignStats, len(campaigns))
	for i, c := range campaigns {
		out[i] = statsByID[c.ID]
	}
	return campaigns, out, nil
}

type int64Slice []int64

func (a int64Slice) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, v := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatInt(v, 10))
	}
	b.WriteByte('}')
	return b.String(), nil
}
