-- name: GetUserStars :one
SELECT stars FROM rating
WHERE username = $1;