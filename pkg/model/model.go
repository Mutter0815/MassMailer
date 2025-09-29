package model

type CreateCampaignReq struct {
	Name        string `json:"name"`
	Body        string `json:"body"`
	ScheduledAt string `json:"scheduled_at"`
	Recipients  string `json:"recipients"`
}

type SendJob struct {
	CampaignID int64  `json:"campaign_id"`
	MessageId  int64  `json:"message_id"`
	Address    string `json:"address"`
	Body       string `json:"body"`
}
