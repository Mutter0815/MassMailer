package campaign

import "time"

type CreateCampaignResp struct {
	ID int64 `json:"id"`
}

type CreateCampaignReq struct {
	Name        string    `json:"name"        binding:"required"`
	Body        string    `json:"body"        binding:"required"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
	Recipients  []string  `json:"recipients"  binding:"required,min=1,dive,required"`
}

type JobMessage struct {
	CampaignID  int64  `json:"campaign_id"`
	RecipientID int64  `json:"recipient_id"`
	Address     string `json:"address"`
}

type Campaign struct {
	ID          int64
	Name        string
	Body        string
	ScheduledAt time.Time
	Status      string
	CreatedAt   time.Time
}
