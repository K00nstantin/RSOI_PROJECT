-- +goose Up 
CREATE TABLE library
(
    id          SERIAL PRIMARY KEY,
    library_uid uuid UNIQUE  NOT NULL,
    name        VARCHAR(80)  NOT NULL,
    city        VARCHAR(255) NOT NULL,
    address     VARCHAR(255) NOT NULL
);

CREATE TABLE books
(
    id        SERIAL PRIMARY KEY,
    book_uid  uuid UNIQUE  NOT NULL,
    name      VARCHAR(255) NOT NULL,
    author    VARCHAR(255),
    genre     VARCHAR(255),
    condition VARCHAR(20) DEFAULT 'EXCELLENT'
        CHECK (condition IN ('EXCELLENT', 'GOOD', 'BAD'))
);

CREATE TABLE library_books
(
    book_id         INT REFERENCES books (id),
    library_id      INT REFERENCES library (id),
    available_count INT NOT NULL
);

-- +goose Down
DROP TABLE library;
DROP TABLE books;
DROP TABLE library_books;
