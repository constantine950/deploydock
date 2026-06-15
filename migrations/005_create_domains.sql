CREATE TYPE ssl_status AS ENUM ('pending', 'active', 'failed');

CREATE TABLE domains (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id      UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    hostname    VARCHAR(512) NOT NULL UNIQUE,
    ssl_status  ssl_status NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_domains_app_id ON domains(app_id);
CREATE INDEX idx_domains_hostname ON domains(hostname);