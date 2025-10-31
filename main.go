package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"` // В реальном приложении хранить хеш!
	Status   string `json:"status"`   // "admin" или "user"
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Status   string `json:"status,omitempty"` // По умолчанию "user"
}

type Claims struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
	jwt.RegisteredClaims
}

var (
	db     *sql.DB
	jwtKey = []byte("my_secret_key_2024")
)

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./products.db")
	if err != nil {
		log.Fatal(err)
	}

	// Создаем таблицы
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			category TEXT NOT NULL,
			price REAL NOT NULL,
			quantity INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'user'
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Заполняем тестовыми данными
	seedData()
}

func seedData() {
	// Проверяем продукты
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil || count == 0 {
		products := []Product{
			{Name: "Яблоки", Category: "Фрукты", Price: 89.99, Quantity: 100},
			{Name: "Молоко", Category: "Молочные продукты", Price: 75.50, Quantity: 50},
			{Name: "Хлеб", Category: "Выпечка", Price: 45.00, Quantity: 30},
		}

		for _, p := range products {
			db.Exec(
				"INSERT INTO products (name, category, price, quantity) VALUES (?, ?, ?, ?)",
				p.Name, p.Category, p.Price, p.Quantity,
			)
		}
	}

	// Проверяем пользователей
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil || count == 0 {
		users := []User{
			{Name: "Admin User", Email: "admin@mail.ru", Password: "admin123", Status: "admin"},
			{Name: "Regular User", Email: "user@mail.ru", Password: "user123", Status: "user"},
			{Name: "Cat User", Email: "cat@bsu.ru", Password: "87654321", Status: "user"},
		}

		for _, u := range users {
			db.Exec(
				"INSERT INTO users (name, email, password, status) VALUES (?, ?, ?, ?)",
				u.Name, u.Email, u.Password, u.Status,
			)
		}
	}
}

// Middleware для JWT аутентификации
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(&w)

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Добавляем данные пользователя в контекст запроса
		ctx := context.WithValue(r.Context(), "userClaims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// Middleware для проверки статуса администратора
func adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value("userClaims").(*Claims)
		if !ok {
			http.Error(w, "Unable to get user claims", http.StatusInternalServerError)
			return
		}

		if claims.Status != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// Логин endpoint
func loginHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверяем пользователя в БД
	var user User
	err := db.QueryRow(
		"SELECT id, name, email, password, status FROM users WHERE email = ? AND password = ?",
		req.Email, req.Password,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Status)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Создаем JWT токен
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Status: user.Status,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Возвращаем токен
	json.NewEncoder(w).Encode(map[string]string{
		"token":   tokenString,
		"message": "Login successful",
		"name":    user.Name,
		"email":   user.Email,
		"status":  user.Status,
	})
}

// Регистрация endpoint
func registerHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Println("Registration request received")

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("Registration attempt for: %s (%s)", req.Name, req.Email)

	// Всегда устанавливаем статус "user" для новых пользователей
	req.Status = "user"

	// Проверяем, не существует ли уже пользователь с таким email
	var existingUser int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", req.Email).Scan(&existingUser)
	if err != nil {
		log.Printf("Database error checking email: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if existingUser > 0 {
		log.Printf("Email already exists: %s", req.Email)
		http.Error(w, "User with this email already exists", http.StatusConflict)
		return
	}

	// Создаем нового пользователя
	result, err := db.Exec(
		"INSERT INTO users (name, email, password, status) VALUES (?, ?, ?, ?)",
		req.Name, req.Email, req.Password, req.Status,
	)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting user ID: %v", err)
		http.Error(w, "Error getting user ID", http.StatusInternalServerError)
		return
	}

	log.Printf("User created successfully with ID: %d", id)

	// Автоматически логиним пользователя после регистрации
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: int(id),
		Name:   req.Name,
		Email:  req.Email,
		Status: req.Status,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	log.Printf("Registration successful for: %s", req.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":   tokenString,
		"message": "Registration successful",
		"name":    req.Name,
		"email":   req.Email,
		"status":  req.Status,
	})
}

// GET /api/profile - получить профиль пользователя
func profileHandler(w http.ResponseWriter, r *http.Request) {
	authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value("userClaims").(*Claims)
		if !ok {
			http.Error(w, "Unable to get user claims", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id": claims.UserID,
			"name":    claims.Name,
			"email":   claims.Email,
			"status":  claims.Status,
		})
	})(w, r)
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

// POST /api/products - создать новый продукт (только для админов)
func createProductHandler(w http.ResponseWriter, r *http.Request) {
	adminMiddleware(func(w http.ResponseWriter, r *http.Request) {
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
	})(w, r)
}

// PUT /api/products/{id} - обновить продукт (только для админов)
func updateProductHandler(w http.ResponseWriter, r *http.Request) {
	adminMiddleware(func(w http.ResponseWriter, r *http.Request) {
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
	})(w, r)
}

// DELETE /api/products/{id} - удалить продукт (только для админов)
func deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	adminMiddleware(func(w http.ResponseWriter, r *http.Request) {
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
	})(w, r)
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

	// Public routes
	router.HandleFunc("/api/login", loginHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/register", registerHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/products", getProductsHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/products/{id}", getProductHandler).Methods("GET", "OPTIONS") // ДОБАВЬТЕ ЭТУ СТРОКУ
	router.HandleFunc("/api/products/category/{category}", getProductsByCategoryHandler).Methods("GET", "OPTIONS")

	// Protected routes
	router.HandleFunc("/api/profile", profileHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/products", createProductHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/products/{id}", updateProductHandler).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/products/{id}", deleteProductHandler).Methods("DELETE", "OPTIONS")

	// Статический файл для простого фронтенда
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	fmt.Println("Server started on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("GET  /api/products")
	fmt.Println("GET  /api/products/{id}")
	fmt.Println("POST /api/products (admin)")
	fmt.Println("PUT  /api/products/{id} (admin)")
	fmt.Println("DELETE /api/products/{id} (admin)")

	log.Fatal(http.ListenAndServe(":8080", router))
}
