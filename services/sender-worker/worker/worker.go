package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/Mutter0815/MassMailer/internal/campaign"
	"github.com/Mutter0815/MassMailer/internal/store"
	"github.com/Mutter0815/MassMailer/pkg/rmq"
)

var errTemp = errors.New("temporary send error")

// имитация отправки (здесь будет SMTP)
func simulateSend(address, body string) error {
	if rand.Float64() < 0.85 {
		return nil
	}
	return errTemp
}

type Worker struct {
	Store *store.Store
	Cons  *rmq.Consumer
}

func New(st *store.Store, cons *rmq.Consumer) *Worker {
	return &Worker{Store: st, Cons: cons}
}

func (w *Worker) Run(ctx context.Context, db *sql.DB) error {
	msgs, err := w.Cons.Consume()
	if err != nil {
		return err
	}
	log.Println("sender-worker started; waiting for jobs...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				return nil
			}
			var job campaign.JobMessage
			if err := json.Unmarshal(d.Body, &job); err != nil {
				log.Println("bad message:", err)
				_ = d.Ack(false)
				continue
			}

			ctx1, cancel1 := context.WithTimeout(ctx, 5*time.Second)
			body, err := w.Store.GetCampaignBody(ctx1, db, job.CampaignID)
			cancel1()
			if err != nil {
				log.Println("db error:", err)
				_ = d.Nack(false, true)
				continue
			}

			if err := simulateSend(job.Address, body); err != nil {
				log.Println("send failed:", err)

				ctx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
				_ = w.Store.MarkMessageFailed(ctx2, db, job.CampaignID, job.RecipientID, err.Error())
				cancel2()

				retries := 0
				if v, ok := d.Headers["x-retries"]; ok {
					if i, ok2 := v.(int32); ok2 {
						retries = int(i)
					}
				}
				if retries < 3 {
					d.Headers["x-retries"] = int32(retries + 1)
					delay := time.Duration(math.Pow(2, float64(retries))) * time.Second
					time.Sleep(delay)
					_ = d.Nack(false, true)
				} else {
					_ = d.Ack(false)
				}
				continue
			}

			ctx3, cancel3 := context.WithTimeout(ctx, 5*time.Second)
			_ = w.Store.MarkMessageSent(ctx3, db, job.CampaignID, job.RecipientID)
			cancel3()

			_ = d.Ack(false)
		}
	}
}
