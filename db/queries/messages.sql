-- name: SaveMessage :one
INSERT INTO messages (session_id, from_jid, to_jid, content, message_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetMessagesBySession :many
SELECT * FROM messages 
WHERE session_id = $1 
ORDER BY sent_at ASC;

-- name: GetRecentMessagesByUserID :many
SELECT m.* FROM messages m
JOIN chat_sessions cs ON m.session_id = cs.id
WHERE cs.user_id = $1
ORDER BY m.sent_at DESC
LIMIT $2;

-- name: GetMessageByID :one
SELECT * FROM messages WHERE id = $1;

-- name: GetMessagesBySessionPaginated :many
SELECT * FROM messages
WHERE session_id = $1
ORDER BY sent_at DESC
LIMIT $2 OFFSET $3;

-- name: CountMessagesBySession :one
SELECT COUNT(*) FROM messages WHERE session_id = $1;

-- name: GetLatestMessageBySession :one
SELECT * FROM messages 
WHERE session_id = $1
ORDER BY sent_at DESC
LIMIT 1;

-- name: GetMessagesSince :many
SELECT m.*, cs.user_id FROM messages m
JOIN chat_sessions cs ON m.session_id = cs.id
WHERE m.sent_at > $1
ORDER BY m.sent_at ASC;