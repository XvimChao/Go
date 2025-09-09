package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type Product struct {
	ID       int
	Name     string
	Category string
	Price    float64
	Quantity int
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./products.db")
	if err != nil {
		log.Fatalf("Ошибка открытия базы данных: %v", err)
	}

	// Создаем таблицу
	createTable := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		price REAL NOT NULL,
		quantity INTEGER NOT NULL
	);
	`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatalf("Ошибка создания таблицы: %v", err)
	}
}

func createProduct() {
	var name, category string
	var price float64
	var quantity int

	fmt.Print("Название продукта: ")
	fmt.Scanln(&name)
	fmt.Print("Категория: ")
	fmt.Scanln(&category)
	fmt.Print("Цена: ")
	fmt.Scanln(&price)
	fmt.Print("Количество: ")
	fmt.Scanln(&quantity)

	stmt, err := db.Prepare("INSERT INTO products(name, category, price, quantity) VALUES(?, ?, ?, ?)")
	if err != nil {
		log.Printf("Ошибка подготовки запроса: %v", err)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(name, category, price, quantity)
	if err != nil {
		log.Printf("Ошибка создания продукта: %v", err)
		return
	}

	id, _ := result.LastInsertId()
	fmt.Printf("Продукт создан успешно! ID: %d\n", id)
}

func getAllProducts() {

	rows, err := db.Query("SELECT id, name, category, price, quantity FROM products ORDER BY id")
	if err != nil {
		log.Printf("Ошибка выполнения запроса: %v", err)
		return
	}
	defer rows.Close()

	var found bool
	for rows.Next() {
		var p Product
		err = rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price, &p.Quantity)
		if err != nil {
			log.Printf("Ошибка чтения данных: %v", err)
			continue
		}
		fmt.Printf("%d: %s (%s) - %.2f руб. (%d шт.)\n",
			p.ID, p.Name, p.Category, p.Price, p.Quantity)
		found = true
	}

	if !found {
		fmt.Println("Продукты не найдены")
	}
}

func updateProduct() {
	var id int
	fmt.Print("Введите ID продукта для обновления: ")
	fmt.Scanln(&id)

	// Сначала проверяем существование продукта
	var existing Product
	err := db.QueryRow("SELECT id, name, category, price, quantity FROM products WHERE id = ?", id).
		Scan(&existing.ID, &existing.Name, &existing.Category, &existing.Price, &existing.Quantity)
	if err != nil {
		fmt.Printf("Продукт с ID %d не найден\n", id)
		return
	}

	fmt.Printf("Текущие данные: %s (%s) - %.2f руб. (%d шт.)\n",
		existing.Name, existing.Category, existing.Price, existing.Quantity)

	var name, category string
	var price float64
	var quantity int

	fmt.Print("Новое название (Enter для сохранения текущего): ")
	fmt.Scanln(&name)
	if name == "" {
		name = existing.Name
	}

	fmt.Print("Новая категория (Enter для сохранения текущей): ")
	fmt.Scanln(&category)
	if category == "" {
		category = existing.Category
	}

	fmt.Print("Новая цена (0 для сохранения текущей): ")
	fmt.Scanln(&price)
	if price == 0 {
		price = existing.Price
	}

	fmt.Print("Новое количество (0 для сохранения текущего): ")
	fmt.Scanln(&quantity)
	if quantity == 0 {
		quantity = existing.Quantity
	}

	stmt, err := db.Prepare("UPDATE products SET name=?, category=?, price=?, quantity=? WHERE id=?")
	if err != nil {
		log.Printf("Ошибка подготовки запроса: %v", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, category, price, quantity, id)
	if err != nil {
		log.Printf("Ошибка обновления продукта: %v", err)
		return
	}

	fmt.Println("Продукт успешно обновлен")
}

func deleteProduct() {
	var id int
	fmt.Print("Введите ID продукта для удаления: ")
	fmt.Scanln(&id)

	// Проверяем существование продукта
	var name string
	err := db.QueryRow("SELECT name FROM products WHERE id = ?", id).Scan(&name)
	if err != nil {
		fmt.Printf("Продукт с ID %d не найден\n", id)
		return
	}

	stmt, err := db.Prepare("DELETE FROM products WHERE id=?")
	if err != nil {
		log.Printf("Ошибка подготовки запроса: %v", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		log.Printf("Ошибка удаления продукта: %v", err)
		return
	}

	fmt.Printf("Продукт '%s' успешно удален\n", name)
}

func searchProducts() {
	var search string
	fmt.Print("Введите название или категорию для поиска: ")
	fmt.Scanln(&search)

	rows, err := db.Query(`
		SELECT id, name, category, price, quantity 
		FROM products 
		WHERE name LIKE ? OR category LIKE ? 
		ORDER BY id
	`, "%"+search+"%", "%"+search+"%")
	if err != nil {
		log.Printf("Ошибка выполнения запроса: %v", err)
		return
	}
	defer rows.Close()

	var found bool
	fmt.Printf("\nРезультаты поиска для '%s':\n", search)
	for rows.Next() {
		var p Product
		err = rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price, &p.Quantity)
		if err != nil {
			log.Printf("Ошибка чтения данных: %v", err)
			continue
		}
		fmt.Printf("%d: %s (%s) - %.2f руб. (%d шт.)\n",
			p.ID, p.Name, p.Category, p.Price, p.Quantity)
		found = true
	}

	if !found {
		fmt.Println("Продукты не найдены")
	}
}

func showMenu() {
	fmt.Println("1. Показать все продукты")
	fmt.Println("2. Создать новый продукт")
	fmt.Println("3. Обновить продукт")
	fmt.Println("4. Удалить продукт")
	fmt.Println("5. Поиск продуктов")
	fmt.Println("6. Выход")
	fmt.Print("Выберите действие: ")
}

func main() {
	initDB()
	defer db.Close()

	for {
		showMenu()

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			getAllProducts()
		case 2:
			createProduct()
		case 3:
			updateProduct()
		case 4:
			deleteProduct()
		case 5:
			searchProducts()
		case 6:
			fmt.Println("Выход из программы...")
			return
		default:
			fmt.Println("Неверный выбор. Попробуйте снова.")
		}
	}
}
