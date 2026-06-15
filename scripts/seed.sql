-- Seed data for development
-- Run with: make seed

-- Test user (password: "password123" - bcrypt hash)
INSERT INTO users (id, name, email, password) VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'Constantine',
    'constantine@deploydock.dev',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'
) ON CONFLICT DO NOTHING;

-- Test app 1: Node.js app
INSERT INTO apps (id, user_id, name, slug, repo_url, branch, runtime, status) VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'My Node App',
    'my-node-app',
    'https://github.com/constantine950/sample-node-app',
    'main',
    'node',
    'idle'
) ON CONFLICT DO NOTHING;

-- Test app 2: Python app
INSERT INTO apps (id, user_id, name, slug, repo_url, branch, runtime, status) VALUES (
    'b0000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000001',
    'My Python App',
    'my-python-app',
    'https://github.com/constantine950/sample-python-app',
    'main',
    'python',
    'idle'
) ON CONFLICT DO NOTHING;

-- Sample deployment for app 1
INSERT INTO deployments (id, app_id, commit_sha, commit_message, status, image_tag) VALUES (
    'c0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000001',
    'abc1234',
    'Initial commit',
    'live',
    'deploydock/b0000000-0000-0000-0000-000000000001:c0000000-0000-0000-0000-000000000001'
) ON CONFLICT DO NOTHING;

-- Sample env vars for app 1
INSERT INTO env_vars (app_id, key, value) VALUES
    ('b0000000-0000-0000-0000-000000000001', 'NODE_ENV', 'production'),
    ('b0000000-0000-0000-0000-000000000001', 'PORT', '3000')
ON CONFLICT DO NOTHING;

SELECT 'Seed complete' AS status;