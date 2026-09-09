#!/bin/sh
set -e

# Если DB_URL не задана – пропускаем всё, связанное с БД
if [ -z "$DB_URL" ]; then
    echo "DB_URL not set, skipping database setup and migrations."
    exec "$@"
fi

# Далее – только если DB_URL задана
echo "Waiting for database at $DB_URL ..."
HOST=$(echo "$DB_URL" | sed -n 's/.*@\([^:]*\):.*/\1/p')
USER=$(echo "$DB_URL" | sed -n 's/.*\/\/\([^:]*\):.*/\1/p')
until pg_isready -h "$HOST" -U "$USER" > /dev/null 2>&1; do
    echo "Postgres is unavailable - sleeping"
    sleep 2
done
echo "Database is ready."

echo "Running migrations..."
goose -dir /migrations postgres "$DB_URL" up

# Проверяем и выполняем seed (если таблица пуста)
TABLE_EXISTS=$(psql "$DB_URL" -t -c "SELECT to_regclass('public.library');" | xargs)
if [ "$TABLE_EXISTS" = "library" ]; then
    COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM library;" | xargs)
    if [ "$COUNT" -eq 0 ] && [ -f /seed.sql ]; then
        echo "Seeding test data..."
        psql "$DB_URL" -f /seed.sql
    else
        echo "Seed data already present or seed.sql missing, skipping."
    fi
else
    echo "Table 'library' does not exist, skipping seed."
fi

# Запускаем приложение
exec "$@"