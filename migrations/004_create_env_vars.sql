CREATE TABLE env_vars (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id      UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key         VARCHAR(255) NOT NULL,
    value       TEXT NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_env_vars_app_key UNIQUE (app_id, key)
);

CREATE INDEX idx_env_vars_app_id ON env_vars(app_id);