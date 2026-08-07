-include .env
export

MIGRATIONS_LIBRARY    = postgres/schema/library_service
MIGRATIONS_RESERVATION = postgres/schema/reservation_service
MIGRATIONS_RATING     = postgres/schema/rating_service

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