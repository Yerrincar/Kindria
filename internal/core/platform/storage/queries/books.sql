-- name: ListBooks :many
SELECT title, author, file_name, bookPath, rating, genres FROM books ORDER BY title;

-- name: InsertBooks :many
INSERT INTO books (title, author, description, genres, language, file_name, bookPath, rating) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateRating :exec
UPDATE books SET rating = ? WHERE title = ?;

-- name: SelectFileNames :many
SELECT file_name FROM books;

-- name: SelectBookPath :one 
SELECT bookPath FROM books WHERE file_name = ?;

-- name: CheckBookExists :one
SELECT COUNT(*) FROM books WHERE file_name = ?;

-- name: SelectAllBooks :many 
SELECT * FROM books ORDER BY title;

