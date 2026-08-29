-- name: GetAllLibraries :many
SELECT * FROM library
WHERE ($1::text IS NULL OR city = $1);

-- name: GetLibrary :one
SELECT * FROM library
WHERE library_uid = $1;  

-- name: GetLibraryBooks :many
SELECT *
FROM library_books lb
JOIN library l ON lb.library_id = l.id
JOIN books b ON lb.book_id = b.id
WHERE l.library_uid = $1;  

-- name: GetBook :one
SELECT * FROM books 
WHERE book_uid = $1;

-- name: DecreaseBookCount :one
UPDATE library_books
SET available_count = available_count - 1
WHERE library_id = (SELECT id FROM library WHERE library_uid = $1)
  AND book_id = (SELECT id FROM books WHERE book_uid = $2)
  AND available_count > 0
  RETURNING *;

-- name: ChangeBookCondition :exec
UPDATE books
SET condition = $1
WHERE book_uid = $2;