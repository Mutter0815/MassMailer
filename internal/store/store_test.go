package store

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInsertCampaign_WithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := New(db)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO campaigns (name,body,scheduled_at,status)
		VALUES ($1,$2,$3,'queued') RETURNING id
	`)).
		WithArgs("n", "b", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	mock.ExpectCommit()

	var id int64
	err = s.WithTx(ctx, func(tx *sql.Tx) error {
		var e error
		id, e = s.InsertCampaign(ctx, tx, "n", "b", "2025-10-02T12:00:00Z")
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != 7 {
		t.Fatalf("want id=7, got %d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInsertRecipient_And_Message(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := New(db)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO recipients (campaign_id, address)
		VALUES ($1,$2) RETURNING id
	`)).
		WithArgs(int64(7), "a@x.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(101))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO messages (campaign_id, recipient_id, status)
		VALUES ($1,$2,'pending')
	`)).
		WithArgs(int64(7), int64(101)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = s.WithTx(ctx, func(tx *sql.Tx) error {
		rid, e := s.InsertRecipient(ctx, tx, 7, "a@x.com")
		if e != nil {
			return e
		}
		return s.InsertMessagePending(ctx, tx, 7, rid)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
