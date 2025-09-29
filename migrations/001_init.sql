
CREATE TABLE  IF NOT EXISTS campaigns(
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    body TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS recipients(
    id BIGSERIAL PRIMARY KEY,
    campaigns_id BIGINT NOT NULL REFERENCES campaigns(id)ON DELETE CASCADE,
    address TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS messages(
    if BIGSERIAL PRIMARY KEY,
    campaigns_id BIGINT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    recipient_id  BIGINT NOT NULL REFERENCES recipients(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMPTZ,
    last_error TEXT,
);