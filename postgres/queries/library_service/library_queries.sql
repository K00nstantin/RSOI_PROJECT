-- name: GetAllLibraries :many
SELECT * FROM library
WHERE ($1::text IS NULL OR city = $1);