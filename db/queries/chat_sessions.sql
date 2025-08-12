-- name: CreateChatSession :one
INSERT INTO chat_sessions (user_id)
VALUES ($1)
RETURNING *;

-- name: GetChatSessionByID :one
SELECT * FROM chat_sessions WHERE id = $1;

-- name: GetActiveSessionByUserID :one
SELECT * FROM chat_sessions 
WHERE user_id = $1 AND status = 'active'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetSessionsByUserID :many
SELECT * FROM chat_sessions 
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateSessionStatus :exec
UPDATE chat_sessions SET status = $1, updated_at = NOW() WHERE id = $2;

-- name: GetActiveSessions :many
SELECT cs.*, u.email, u.xmpp_jid
FROM chat_sessions cs
JOIN users u ON cs.user_id = u.id
WHERE cs.status = 'active'
ORDER BY cs.created_at DESC;

-- name: CloseChatSession :exec
UPDATE chat_sessions SET status = 'closed', updated_at = NOW() WHERE id = $1;