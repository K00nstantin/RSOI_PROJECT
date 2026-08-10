-- name: GetAllLibraries :many
SELECT * FROM library
WHERE ($1::text IS NULL OR city = $1);

-- name: GetLibraryBooks :many
SELECT *
FROM library_books lb
JOIN library l ON lb.library_id = l.id
JOIN books b ON lb.book_id = b.id
WHERE l.library_uid = $1;  