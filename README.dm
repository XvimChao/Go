Для запуска нужен компилятор gcc

Инициализация модуля
go mod init

Для запуска программы:
go run main.go

Тема: Продукты

Таблица Products:

Название        Тип                       Описание
--------------------------------------------------------
id          INTEGER PRIMARY KEY      Уникальный идентификатор
name        TEXT NOT NULL           Название продукта
category    TEXT NOT NULL           Категория продукта
price       REAL NOT NULL           Цена за единицу(кило)
quantity    INTEGER NOT NULL        Количество на складе

Таблица users
Название	Тип	Описание
--------------------------------------------------------
id	        INTEGER PRIMARY KEY	    Уникальный идентификатор
name	    TEXT NOT NULL	        Имя пользователя
email	    TEXT UNIQUE NOT NULL	Email пользователя
password	TEXT NOT NULL	        Пароль
status	    TEXT NOT NULL	        Статус (admin/user)

Публичные endpoints (не требуют авторизации):

POST /api/login - Аутентификация пользователя

POST /api/register - Регистрация нового пользователя

GET /api/products - Получить все продукты

GET /api/products/{id} - Получить продукт по ID

GET /api/products/category/{category} - Поиск продуктов по категории


Защищенные endpoints (требуют JWT токен):

POST /api/products - Создать новый продукт (только admin)

PUT /api/products/{id} - Обновить продукт (только admin)

DELETE /api/products/{id} - Удалить продукт (только admin)



Антропов Ардан гр.01321


