package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Mutter0815/MassMailer/internal/campaign"
	"github.com/Mutter0815/MassMailer/internal/store"
	"github.com/Mutter0815/MassMailer/pkg/logx"
	"github.com/Mutter0815/MassMailer/pkg/metrics"
	"github.com/Mutter0815/MassMailer/pkg/rmq"
	"github.com/gin-gonic/gin"
)

type storeAPI interface {
	WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error
	InsertCampaign(ctx context.Context, tx *sql.Tx, name, body string, scheduledAt any) (int64, error)
	InsertRecipient(ctx context.Context, tx *sql.Tx, campaignID int64, address string) (int64, error)
	InsertMessagePending(ctx context.Context, tx *sql.Tx, campaignID, recipientID int64) error
	GetCampaign(ctx context.Context, id int64) (store.CampaignRow, error)
	GetCampaignStats(ctx context.Context, id int64) (store.CampaignStats, error)
	ListCampaigns(ctx context.Context, limit, offset int) ([]store.CampaignRow, []store.CampaignStats, error)
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
		payload, err := json.Marshal(job)
		if err != nil {
			logx.L().Errorw("job_marshal_error", "campaign_id", campaignID, "recipient_id", r.id, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "publish error"})
			return
		}
		if err := h.Pub.PublishJSON(ctxPub, payload); err != nil {
			logx.L().Errorw("publish_job_error", "campaign_id", campaignID, "recipient_id", r.id, "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "queue unavailable"})
			return
		}
		metrics.PublishedJobsTotal.Inc()
	}

	c.JSON(http.StatusOK, campaign.CreateCampaignResp{ID: campaignID})
}

func (h *Handlers) ListCampaigns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, stats, err := h.Store.ListCampaigns(ctx, limit, offset)
	if err != nil {
		logx.L().Errorw("list_campaigns_error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}

	out := make([]campaign.CampaignListItem, 0, len(rows))
	for i, r := range rows {
		item := campaign.CampaignListItem{
			ID:          r.ID,
			Name:        r.Name,
			ScheduledAt: r.ScheduledAt,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
		}
		item.Stats.Total = stats[i].Total
		item.Stats.Pending = stats[i].Pending
		item.Stats.Sent = stats[i].Sent
		item.Stats.Failed = stats[i].Failed
		out = append(out, item)
	}

	c.JSON(http.StatusOK, out)
}

func (h *Handlers) GetCampaign(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	camp, err := h.Store.GetCampaign(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}

	stats, err := h.Store.GetCampaignStats(ctx, id)
	if err != nil {
		logx.L().Errorw("get_campaign_stats_error", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stats error"})
		return
	}

	resp := campaign.CampaignDetails{
		ID:          camp.ID,
		Name:        camp.Name,
		Body:        camp.Body,
		ScheduledAt: camp.ScheduledAt,
		Status:      camp.Status,
		CreatedAt:   camp.CreatedAt,
	}
	resp.Stats.Total = stats.Total
	resp.Stats.Pending = stats.Pending
	resp.Stats.Sent = stats.Sent
	resp.Stats.Failed = stats.Failed

	c.JSON(http.StatusOK, resp)
}
