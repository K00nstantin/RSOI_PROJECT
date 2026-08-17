-include .env
export

MIGRATIONS_LIBRARY    = postgres/schema/library_service
MIGRATIONS_RESERVATION = postgres/schema/reservation_service
MIGRATIONS_RATING     = postgres/schema/rating_service

DB_USER     ?= postgres
DB_HOST     ?= localhost
DB_PORT     ?= 5432

check-dsn:
	@if [ -z $(LIBRARIES_DB_URL) ]; then echo "Error: DSN_LIBRARY not set. Please define it in .env or export it."; exit 1; fi
	@if [ -z $(RESERVATIONS_DB_URL) ]; then echo "Error: DSN_RESERVATION not set. Please define it in .env or export it."; exit 1; fi
	@if [ -z $(RATINGS_DB_URL) ]; then echo "Error: DSN_RATING not set. Please define it in .env or export it."; exit 1; fi

.PHONY: up-all down-all check-dsn

up-all: check-dsn
	goose -dir $(MIGRATIONS_LIBRARY) postgres $(LIBRARIES_DB_URL) up
	goose -dir $(MIGRATIONS_RESERVATION) postgres $(RESERVATIONS_DB_URL) up
	goose -dir $(MIGRATIONS_RATING) postgres $(RATINGS_DB_URL) up

down-all: check-dsn
	goose -dir $(MIGRATIONS_LIBRARY) postgres $(LIBRARIES_DB_URL) down
	goose -dir $(MIGRATIONS_RESERVATION) postgres $(RESERVATIONS_DB_URL) down
	goose -dir $(MIGRATIONS_RATING) postgres $(RATINGS_DB_URL) down

.PHONY: clean-db
clean-db:
	psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -d libraries -c \
		"TRUNCATE TABLE library_books, books, library RESTART IDENTITY CASCADE;"
	psql -U $(DB_USER) -h $(DB_HOSTALTER USER postgres WITH PASSWORD 'новый_пароль';) -p $(DB_PORT) -d reservations -c \
		"TRUNCATE TABLE reservation RESTART IDENTITY CASCADE;"
	psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -d ratings -c \
		"TRUNCATE TABLE rating RESTART IDENTITY CASCADE;"

.PHONY: seed-db
seed-db:
	psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -d libraries -c \
		"INSERT INTO library (library_uid, name, city, address) \
		 VALUES ('83575e12-7ce0-48ee-9931-51919ff3c9ee', \
				 'Библиотека имени 7 Непьющих', \
				 'Москва', \
				 '2-я Бауманская ул., д.5, стр.1');"
	psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -d libraries -c \
		"INSERT INTO books (book_uid, name, author, genre, condition) \
		 VALUES ('f7cdc58f-2caf-4b15-9727-f89dcc629b27', \
				 'Краткий курс C++ в 7 томах', \
				 'Бьерн Страуструп', \
				 'Научная фантастика', \
				 'EXCELLENT');"
	psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -d libraries -c \
		"INSERT INTO library_books (library_id, book_id, available_count) \
		 SELECT l.id, b.id, 1 \
		 FROM library l, books b \
		 WHERE l.library_uid = '83575e12-7ce0-48ee-9931-51919ff3c9ee' \
		   AND b.book_uid   = 'f7cdc58f-2caf-4b15-9727-f89dcc629b27';"
