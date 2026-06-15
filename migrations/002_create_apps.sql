CREATE TYPE app_runtime AS ENUM ('node', 'python', 'go', 'static');
CREATE TYPE app_status AS ENUM ('idle', 'building', 'deploying', 'live', 'failed');

CREATE TABLE apps (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL UNIQUE,
    repo_url    VARCHAR(1024) NOT NULL,
    branch      VARCHAR(255) NOT NULL DEFAULT 'main',
    runtime     app_runtime,
    status      app_status NOT NULL DEFAULT 'idle',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_apps_user_id ON apps(user_id);
CREATE INDEX idx_apps_slug ON apps(slug);