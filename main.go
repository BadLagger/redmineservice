package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Config структура для конфигурации
type Config struct {
	RedmineURL string
	Port       string
}

// RedmineError структура для ошибок Redmine
type RedmineError struct {
	Error string `json:"error"`
}

var config = Config{
	RedmineURL: "http://redmine.kbnavis.local", // ваш URL Redmine
	Port:       "9995",
}

func main() {
	// Настройка маршрутов
	http.HandleFunc("/", serveStatic)
	http.HandleFunc("/api/", proxyRedmineAPI)
	http.HandleFunc("/health", healthCheck)

	log.Printf("🚀 Сервер запущен на http://localhost:%s", config.Port)
	log.Printf("📊 Redmine URL: %s", config.RedmineURL)
	log.Printf("📁 Откройте http://localhost:%s в браузере", config.Port)

	err := http.ListenAndServe(":"+config.Port, nil)
	if err != nil {
		log.Fatal("❌ Ошибка запуска сервера:", err)
	}
}

// serveStatic раздает статические файлы
func serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "deadtasks.html")
		return
	}

	// Раздаем другие статические файлы
	http.ServeFile(w, r, "."+r.URL.Path)
}

// proxyRedmineAPI проксирует запросы к Redmine API
func proxyRedmineAPI(w http.ResponseWriter, r *http.Request) {
	// Разрешаем CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Redmine-API-Key")

	if r.Method == "OPTIONS" {
		return
	}

	// Извлекаем API ключ из query параметров
	apiKey := r.URL.Query().Get("key")
	if apiKey == "" {
		sendError(w, "Отсутствует API ключ", http.StatusBadRequest)
		return
	}

	// Формируем URL для Redmine API
	apiPath := strings.TrimPrefix(r.URL.Path, "/api/")
	redmineURL := fmt.Sprintf("%s/%s?%s", config.RedmineURL, apiPath, r.URL.RawQuery)

	log.Printf("📨 Проксируем запрос: %s", apiPath)

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{Timeout: 30 * time.Second}

	// Создаем запрос к Redmine
	req, err := http.NewRequest(r.Method, redmineURL, nil)
	if err != nil {
		sendError(w, fmt.Sprintf("Ошибка создания запроса: %v", err), http.StatusInternalServerError)
		return
	}

	// Добавляем заголовки для Redmine API
	req.Header.Set("X-Redmine-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		sendError(w, fmt.Sprintf("Ошибка подключения к Redmine: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Копируем заголовки ответа
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Копируем статус код
	w.WriteHeader(resp.StatusCode)

	// Копируем тело ответа
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Ошибка копирования ответа: %v", err)
	}
}

// healthCheck проверка здоровья сервера
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "redmine-proxy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// sendError отправляет ошибку в JSON формате
func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(RedmineError{
		Error: message,
	})
}
