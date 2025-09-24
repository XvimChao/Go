package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./products.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		price REAL NOT NULL,
		quantity INTEGER NOT NULL
	);
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	// Заполняем тестовыми данными
	seedData()
}

func seedData() {
	// Проверяем, есть ли уже данные
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	products := []Product{
		{Name: "Яблоки", Category: "Фрукты", Price: 89.99, Quantity: 100},
		{Name: "Молоко", Category: "Молочные продукты", Price: 75.50, Quantity: 50},
		{Name: "Хлеб", Category: "Выпечка", Price: 45.00, Quantity: 30},
		{Name: "Сыр", Category: "Молочные продукты", Price: 250.00, Quantity: 20},
		{Name: "Помидоры", Category: "Овощи", Price: 120.00, Quantity: 40},
	}

	for _, p := range products {
		_, err := db.Exec(
			"INSERT INTO products (name, category, price, quantity) VALUES (?, ?, ?, ?)",
			p.Name, p.Category, p.Price, p.Quantity,
		)
		if err != nil {
			log.Printf("Ошибка при добавлении тестовых данных: %v", err)
		}
	}
}

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// GET /api/products - получить все продукты
func getProductsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	rows, err := db.Query("SELECT id, name, category, price, quantity FROM products")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price, &p.Quantity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// GET /api/products/{id} - получить продукт по ID
func getProductHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var product Product
	err = db.QueryRow(
		"SELECT id, name, category, price, quantity FROM products WHERE id = ?",
		id,
	).Scan(&product.ID, &product.Name, &product.Category, &product.Price, &product.Quantity)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Product not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// POST /api/products - создать новый продукт
func createProductHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var product Product
	err := json.NewDecoder(r.Body).Decode(&product)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := db.Exec(
		"INSERT INTO products (name, category, price, quantity) VALUES (?, ?, ?, ?)",
		product.Name, product.Category, product.Price, product.Quantity,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	product.ID = int(id)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

// PUT /api/products/{id} - обновить продукт
func updateProductHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var product Product
	err = json.NewDecoder(r.Body).Decode(&product)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec(
		"UPDATE products SET name = ?, category = ?, price = ?, quantity = ? WHERE id = ?",
		product.Name, product.Category, product.Price, product.Quantity, id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	product.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// DELETE /api/products/{id} - удалить продукт
func deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM products WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/products/category/{category} - поиск по категории
func getProductsByCategoryHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	category := vars["category"]

	rows, err := db.Query(
		"SELECT id, name, category, price, quantity FROM products WHERE category = ?",
		category,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price, &p.Quantity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func main() {
	initDB()
	defer db.Close()

	router := mux.NewRouter()

	// Роуты API
	router.HandleFunc("/api/products", getProductsHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/products", createProductHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/products/{id}", getProductHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/products/{id}", updateProductHandler).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/products/{id}", deleteProductHandler).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/products/category/{category}", getProductsByCategoryHandler).Methods("GET", "OPTIONS")

	// Статический файл для простого фронтенда
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	fmt.Println("Server started on :8080")
	fmt.Println("API endpoints:")
	fmt.Println("  GET    /api/products")
	fmt.Println("  POST   /api/products")
	fmt.Println("  GET    /api/products/{id}")
	fmt.Println("  PUT    /api/products/{id}")
	fmt.Println("  DELETE /api/products/{id}")
	fmt.Println("  GET    /api/products/category/{category}")

	log.Fatal(http.ListenAndServe(":8080", router))
}