package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"redmineproxy/utils"
	"strings"
	"time"
)

// RedmineError структура для ошибок Redmine
type RedmineError struct {
	Error string `json:"error"`
}

type ProxyController struct {
	log    *utils.Logger
	apiKey string
	url    string
}

func InitProxyController(apiKey, redmineUrl string) *ProxyController {
	return &ProxyController{
		log:    utils.GlobalLogger(),
		apiKey: apiKey,
		url:    redmineUrl,
	}
}

func (c *ProxyController) ProxyRedmineAPI(w http.ResponseWriter, r *http.Request) {
	// Разрешаем CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Redmine-API-Key")

	// Извлекаем API ключ из query параметров
	/*apiKey := r.URL.Query().Get("key")
	if apiKey == "" {
		sendError(w, "Отсутствует API ключ", http.StatusBadRequest)
		return
	}*/

	// Формируем URL для Redmine API
	apiPath := strings.TrimPrefix(r.URL.Path, "/api/")
	redmineURL := fmt.Sprintf("%s/%s?%s", c.url, apiPath, r.URL.RawQuery)

	c.log.Debug("Proxy request: %s", apiPath)

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{Timeout: 30 * time.Second}

	// Создаем запрос к Redmine
	req, err := http.NewRequest(r.Method, redmineURL, nil)
	if err != nil {
		sendError(w, fmt.Sprintf("Create request error: %v", err), http.StatusInternalServerError)
		return
	}

	// Добавляем заголовки для Redmine API
	req.Header.Set("X-Redmine-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		sendError(w, fmt.Sprintf("Error Redmine connection: %v", err), http.StatusBadGateway)
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

// sendError отправляет ошибку в JSON формате
func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(RedmineError{
		Error: message,
	})
}
