-- name: GetUserStars :one
SELECT stars FROM rating
WHERE username = $1;

-- name: UpdateRating :exec 
UPDATE rating
SET stars = stars + $1
WHERE username = $2;
