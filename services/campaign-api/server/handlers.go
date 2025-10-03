package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Mutter0815/MassMailer/internal/campaign"
	"github.com/Mutter0815/MassMailer/internal/store"
	"github.com/Mutter0815/MassMailer/pkg/rmq"
	"github.com/gin-gonic/gin"
)

type storeAPI interface {
	WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error
	InsertCampaign(ctx context.Context, tx *sql.Tx, name, body string, scheduledAt any) (int64, error)
	InsertRecipient(ctx context.Context, tx *sql.Tx, campaignID int64, address string) (int64, error)
	InsertMessagePending(ctx context.Context, tx *sql.Tx, campaignID, recipientID int64) error
}

type publisherAPI interface {
	PublishJSON(ctx context.Context, body []byte) error
}

type storeAdapter struct{ *store.Store }
type publisherAdapter struct{ *rmq.Publisher }

type Handlers struct {
	Store storeAPI
	Pub   publisherAPI
}

func NewHandlers(s *store.Store, pub *rmq.Publisher) *Handlers {
	return &Handlers{Store: &storeAdapter{s}, Pub: &publisherAdapter{pub}}
}

func (h *Handlers) Healthz(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func (h *Handlers) CreateCampaign(c *gin.Context) {
	var req campaign.CreateCampaignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var campaignID int64
	recs := make([]struct {
		id   int64
		addr string
	}, 0, len(req.Recipients))

	err := h.Store.WithTx(ctx, func(tx *sql.Tx) error {
		id, err := h.Store.InsertCampaign(ctx, tx, req.Name, req.Body, req.ScheduledAt)
		if err != nil {
			return err
		}
		campaignID = id

		for _, addr := range req.Recipients {
			rid, err := h.Store.InsertRecipient(ctx, tx, campaignID, addr)
			if err != nil {
				return err
			}
			if err := h.Store.InsertMessagePending(ctx, tx, campaignID, rid); err != nil {
				return err
			}
			recs = append(recs, struct {
				id   int64
				addr string
			}{id: rid, addr: addr})
		}
		return nil
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ctxPub, cancelPub := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPub()

	for _, r := range recs {
		job := campaign.JobMessage{
			CampaignID:  campaignID,
			RecipientID: r.id,
			Address:     r.addr,
		}
		if b, _ := json.Marshal(job); h.Pub.PublishJSON(ctxPub, b) != nil {
			// TODO: лог/метрика
		}
	}

	c.JSON(http.StatusOK, campaign.CreateCampaignResp{ID: campaignID})
}
