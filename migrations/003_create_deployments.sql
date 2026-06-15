CREATE TYPE deployment_status AS ENUM (
    'queued',
    'building',
    'deploying',
    'live',
    'failed',
    'rolled_back'
);

CREATE TABLE deployments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id          UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    commit_sha      VARCHAR(40),
    commit_message  TEXT,
    status          deployment_status NOT NULL DEFAULT 'queued',
    container_id    VARCHAR(255),
    image_tag       VARCHAR(512),
    port            INTEGER,
    error_message   TEXT,
    started_at      TIMESTAMP WITH TIME ZONE,
    finished_at     TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_deployments_app_id ON deployments(app_id);
CREATE INDEX idx_deployments_status ON deployments(status);