package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Mutter0815/MassMailer/internal/campaign"
	"github.com/Mutter0815/MassMailer/internal/store"
	"github.com/Mutter0815/MassMailer/pkg/logx"
	"github.com/Mutter0815/MassMailer/pkg/metrics"
	"github.com/Mutter0815/MassMailer/pkg/rmq"
)

var errTemp = errors.New("temporary send error")

func init() {
	rand.Seed(time.Now().UnixNano())
}

// имитация отправки (здесь будет SMTP/API)
func simulateSend(address, body string) error {
	if rand.Float64() < 0.85 {
		return nil
	}
	return errTemp
}

type Worker struct {
	Store *store.Store
	Cons  *rmq.Consumer
	Pub   *rmq.Publisher
}

func New(st *store.Store, cons *rmq.Consumer, pub *rmq.Publisher) *Worker {
	return &Worker{Store: st, Cons: cons, Pub: pub}
}

func (w *Worker) Run(ctx context.Context, db *sql.DB) error {
	msgs, err := w.Cons.Consume()
	if err != nil {
		return err
	}
	logx.L().Infow("worker_started", "queue", w.Cons.Queue)

	for {
		select {
		case <-ctx.Done():
			logx.L().Infow("worker_stopping")
			return ctx.Err()

		case d, ok := <-msgs:
			if !ok {
				logx.L().Warnw("consumer_channel_closed")
				return nil
			}

			start := time.Now()
			metrics.WorkerJobsConsumed.Inc()

			var job campaign.JobMessage
			if err := json.Unmarshal(d.Body, &job); err != nil {
				logx.L().Warnw("job_unmarshal_error", "error", err)
				_ = d.Ack(false)
				metrics.WorkerProcessDuration.Observe(time.Since(start).Seconds())
				continue
			}
			fields := []any{
				"campaign_id", job.CampaignID,
				"recipient_id", job.RecipientID,
				"address", job.Address,
			}

			ctx1, cancel1 := context.WithTimeout(ctx, 5*time.Second)
			body, err := w.Store.GetCampaignBody(ctx1, db, job.CampaignID)
			cancel1()
			if err != nil {
				logx.L().Errorw("db_get_campaign_body_error", append(fields, "error", err)...)
				_ = d.Nack(false, true)
				metrics.WorkerProcessDuration.Observe(time.Since(start).Seconds())
				continue
			}

			if err := simulateSend(job.Address, body); err != nil {
				logx.L().Infow("send_failed", append(fields, "error", err)...)

				ctx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
				if err := w.Store.MarkMessageFailed(ctx2, db, job.CampaignID, job.RecipientID, err.Error()); err != nil {
					cancel2()
					logx.L().Errorw("db_mark_failed_error", append(fields, "error", err)...)
					metrics.WorkerProcessDuration.Observe(time.Since(start).Seconds())
					_ = d.Nack(false, true)
					continue
				}
				cancel2()

				metrics.WorkerJobsFailed.Inc()

				retries := headerRetries(d.Headers)
				if retries < 3 {
					delay := backoffDelay(retries)
					metrics.WorkerJobRetries.Inc()
					logx.L().Infow("retry_requeue", append(fields, "retries", retries+1, "delay", delay.String())...)
					if err := w.requeueMessage(ctx, d, retries+1, delay); err != nil {
						logx.L().Errorw("retry_publish_error", append(fields, "retries", retries+1, "error", err)...)
						_ = d.Nack(false, true)
					}
				} else {
					logx.L().Warnw("drop_after_retries", append(fields, "retries", retries)...)
					_ = d.Ack(false)
				}

				metrics.WorkerProcessDuration.Observe(time.Since(start).Seconds())
				continue
			}

			ctx3, cancel3 := context.WithTimeout(ctx, 5*time.Second)
			if err := w.Store.MarkMessageSent(ctx3, db, job.CampaignID, job.RecipientID); err != nil {
				cancel3()
				logx.L().Errorw("db_mark_sent_error", append(fields, "error", err)...)
				metrics.WorkerProcessDuration.Observe(time.Since(start).Seconds())
				_ = d.Nack(false, true)
				continue
			}
			cancel3()

			metrics.WorkerJobsSent.Inc()
			metrics.WorkerProcessDuration.Observe(time.Since(start).Seconds())

			logx.L().Infow("send_success", fields...)
			_ = d.Ack(false)
		}
	}
}

func (w *Worker) requeueMessage(ctx context.Context, d amqp.Delivery, retries int, delay time.Duration) error {
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	headers := copyHeaders(d.Headers)
	setHeaderRetries(&headers, retries)

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := w.Pub.PublishJSONWithHeaders(pubCtx, d.Body, headers); err != nil {
		return err
	}

	return d.Ack(false)
}

func headerRetries(h amqp.Table) int {
	if h == nil {
		return 0
	}
	if v, ok := h["x-retries"]; ok {
		switch t := v.(type) {
		case int32:
			return int(t)
		case int64:
			return int(t)
		case int:
			return t
		case uint8:
			return int(t)
		}
	}
	return 0
}

func setHeaderRetries(h *amqp.Table, n int) {
	if *h == nil {
		*h = amqp.Table{}
	}
	(*h)["x-retries"] = int32(n)
}

func backoffDelay(retries int) time.Duration {
	if retries <= 0 {
		return 0
	}
	sec := math.Pow(2, float64(retries-1))
	return time.Duration(sec) * time.Second
}

func copyHeaders(h amqp.Table) amqp.Table {
	if h == nil {
		return amqp.Table{}
	}
	dup := make(amqp.Table, len(h))
	for k, v := range h {
		dup[k] = v
	}
	return dup
}
