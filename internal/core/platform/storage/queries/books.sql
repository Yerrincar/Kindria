-- name: ListBooks :many
SELECT title, author, file_name, bookPath FROM books ORDER BY title;

-- name: InsertBooks :many
INSERT INTO books (title, author, description, genders, language, file_name, bookPath) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: SelectFileNames :many
SELECT file_name FROM books;

-- name: CheckBookExists :one
SELECT COUNT(*) FROM books WHERE file_name = ?;

-- name: SelectAllBooks :many 
SELECT * FROM books ORDER BY title;

