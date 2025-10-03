CREATE TABLE IF NOT EXISTS campaigns (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT        NOT NULL,
    body         TEXT        NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'queued',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT campaigns_status_chk
      CHECK (status IN ('queued','processing','done','failed','canceled'))
);

CREATE TABLE IF NOT EXISTS recipients (
    id          BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    address     TEXT   NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
    id           BIGSERIAL PRIMARY KEY,
    campaign_id  BIGINT       NOT NULL REFERENCES campaigns(id)   ON DELETE CASCADE,
    recipient_id BIGINT       NOT NULL REFERENCES recipients(id)  ON DELETE CASCADE,
    status       TEXT         NOT NULL DEFAULT 'pending',         
    sent_at      TIMESTAMPTZ,
    last_error   TEXT,
    CONSTRAINT messages_status_chk
      CHECK (status IN ('pending','sent','failed'))
);

CREATE INDEX IF NOT EXISTS idx_campaigns_sched     ON campaigns (scheduled_at);
CREATE INDEX IF NOT EXISTS idx_recipients_campaign ON recipients (campaign_id);
CREATE INDEX IF NOT EXISTS idx_messages_campaign   ON messages   (campaign_id);
CREATE INDEX IF NOT EXISTS idx_messages_status     ON messages   (status);

CREATE UNIQUE INDEX IF NOT EXISTS uq_msg_campaign_recipient
  ON messages (campaign_id, recipient_id);
