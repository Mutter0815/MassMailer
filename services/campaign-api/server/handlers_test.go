package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mutter0815/MassMailer/internal/campaign"
	"github.com/Mutter0815/MassMailer/internal/store"
)

type fakeStore struct {
	failTx            bool
	insertCampaignHit bool
	recipientsN       int
	msgsN             int
}

func (f *fakeStore) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if f.failTx {
		return errTest("tx failed")
	}
	return fn(&sql.Tx{})
}

func (f *fakeStore) InsertCampaign(ctx context.Context, tx *sql.Tx, name, body string, scheduledAt any) (int64, error) {
	f.insertCampaignHit = true
	return int64(42), nil
}

func (f *fakeStore) InsertRecipient(ctx context.Context, tx *sql.Tx, campaignID int64, address string) (int64, error) {
	f.recipientsN++
	return int64(100 + f.recipientsN), nil
}

func (f *fakeStore) InsertMessagePending(ctx context.Context, tx *sql.Tx, campaignID, recipientID int64) error {
	f.msgsN++
	return nil
}

func (f *fakeStore) GetCampaign(ctx context.Context, id int64) (store.CampaignRow, error) {
	return store.CampaignRow{
		ID:          id,
		Name:        "stub",
		Body:        "body",
		ScheduledAt: time.Unix(0, 0).UTC(),
		Status:      "queued",
		CreatedAt:   time.Unix(0, 0).UTC(),
	}, nil
}

func (f *fakeStore) GetCampaignStats(ctx context.Context, id int64) (store.CampaignStats, error) {
	return store.CampaignStats{
		Total:   3,
		Pending: 0,
		Sent:    2,
		Failed:  1,
	}, nil
}

func (f *fakeStore) ListCampaigns(ctx context.Context, limit, offset int) ([]store.CampaignRow, []store.CampaignStats, error) {
	rows := []store.CampaignRow{
		{ID: 1, Name: "A", ScheduledAt: time.Unix(0, 0).UTC(), Status: "queued", CreatedAt: time.Unix(0, 0).UTC()},
		{ID: 2, Name: "B", ScheduledAt: time.Unix(0, 0).UTC(), Status: "queued", CreatedAt: time.Unix(0, 0).UTC()},
	}
	stats := []store.CampaignStats{
		{Total: 3, Pending: 0, Sent: 3, Failed: 0},
		{Total: 3, Pending: 1, Sent: 1, Failed: 1},
	}
	return rows, stats, nil
}

type fakePublisher struct{ n int }

func (p *fakePublisher) PublishJSON(ctx context.Context, body []byte) error {
	p.n++
	return nil
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestCreateCampaign_OK(t *testing.T) {
	fs := &fakeStore{}
	fp := &fakePublisher{}
	h := &Handlers{Store: fs, Pub: fp}

	srv := NewHTTPServer(":0", h)
	rr := httptest.NewRecorder()

	body := bytes.NewBufferString(`{
		"name":"Smoke",
		"body":"Hello",
		"scheduled_at":"2025-10-02T12:00:00Z",
		"recipients":["u1@example.com","u2@example.com"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/campaigns", body)
	req.Header.Set("Content-Type", "application/json")

	srv.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rr.Code, rr.Body.String())
	}
	var resp campaign.CreateCampaignResp
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ID != 42 {
		t.Fatalf("want id=42, got %d", resp.ID)
	}
	if !fs.insertCampaignHit {
		t.Fatal("insertCampaign not called")
	}
	if fs.recipientsN != 2 || fs.msgsN != 2 {
		t.Fatalf("want 2 recipients & 2 messages, got %d/%d", fs.recipientsN, fs.msgsN)
	}
	if fp.n != 2 {
		t.Fatalf("want 2 published messages, got %d", fp.n)
	}
}

func TestCreateCampaign_ValidationError(t *testing.T) {
	h := &Handlers{Store: &fakeStore{}, Pub: &fakePublisher{}}
	srv := NewHTTPServer(":0", h)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/campaigns", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")

	srv.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateCampaign_TxError(t *testing.T) {
	fs := &fakeStore{failTx: true}
	h := &Handlers{Store: fs, Pub: &fakePublisher{}}
	srv := NewHTTPServer(":0", h)

	rr := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
                "name":"X","body":"Y",
                "scheduled_at":"2025-10-02T12:00:00Z",
                "recipients":["u@example.com"]
        }`)
	req := httptest.NewRequest(http.MethodPost, "/campaigns", body)
	req.Header.Set("Content-Type", "application/json")

	srv.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestDocsEndpoints(t *testing.T) {
	h := &Handlers{Store: &fakeStore{}, Pub: &fakePublisher{}}
	srv := NewHTTPServer(":0", h)

	t.Run("html", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)

		srv.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "SwaggerUIBundle") {
			t.Fatalf("swagger bundle not rendered: %s", rr.Body.String())
		}
	})

	t.Run("openapi", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/docs/campaign-api/openapi.yaml", nil)

		srv.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "yaml") {
			t.Fatalf("unexpected content type: %s", ct)
		}
		if !strings.Contains(rr.Body.String(), "openapi: 3.0.3") {
			t.Fatalf("unexpected body: %s", rr.Body.String())
		}
	})
}
