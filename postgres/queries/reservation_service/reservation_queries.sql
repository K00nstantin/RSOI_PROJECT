-- name: GetReservations :many
SELECT * FROM reservation
WHERE username = $1;