-- name: GetReservations :many
SELECT * FROM reservation
WHERE username = $1;

-- name: GetActiveReservations :many
SELECT * FROM reservation
WHERE username = $1 AND status = 'RENTED';

-- name: CreateReservation :one
INSERT INTO reservation (reservation_uid, username, book_uid, library_uid, status, start_date, till_date)
VALUES (gen_random_uuid(), $1, $2, $3, 'RENTED', NOW(), $4)
RETURNING reservation_uid, status, start_date, till_date;