CREATE TYPE log_stream AS ENUM ('stdout', 'stderr');

CREATE TABLE build_logs (
    id              BIGSERIAL PRIMARY KEY,
    deployment_id   UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    stream          log_stream NOT NULL DEFAULT 'stdout',
    line            TEXT NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_build_logs_deployment_id ON build_logs(deployment_id);
CREATE INDEX idx_build_logs_created_at ON build_logs(created_at);