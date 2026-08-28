-- name: GetUserByUsername :one 
SELECT * FROM users 
WHERE username = $1;

-- name: InsertUser :exec
INSERT INTO users (username, password_hash, email, role)
VALUES ($1, $2, $3, $4);


