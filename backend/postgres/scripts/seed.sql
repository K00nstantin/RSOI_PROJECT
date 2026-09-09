-- Вставка библиотеки
INSERT INTO library (library_uid, name, city, address)
VALUES (
    '83575e12-7ce0-48ee-9931-51919ff3c9ee',
    'Библиотека имени 7 Непьющих',
    'Москва',
    '2-я Бауманская ул., д.5, стр.1'
) ON CONFLICT (library_uid) DO NOTHING;

-- Вставка книги
INSERT INTO books (book_uid, name, author, genre, condition)
VALUES (
    'f7cdc58f-2caf-4b15-9727-f89dcc629b27',
    'Краткий курс C++ в 7 томах',
    'Бьерн Страуструп',
    'Научная фантастика',
    'EXCELLENT'
) ON CONFLICT (book_uid) DO NOTHING;

-- Связь: вставляем, только если записи существуют
INSERT INTO library_books (book_id, library_id, available_count)
VALUES (1, 1, 2) ;