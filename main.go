package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"redmineproxy/controller"
	"redmineproxy/utils"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	log := utils.GlobalLogger().SetLevel(utils.Debug)
	cfg := utils.CfgLoad("RedmineService")

	log.Info("Application %s starts!", cfg.AppName)
	defer log.Info("Application %s done!", cfg.AppName)

	if len(cfg.HostPort) == 0 {
		log.Critical("HOST_PORT is not defined!")
		return
	}

	if len(cfg.RedmineAddress) == 0 {
		log.Critical("REDMINE_ADDRESS is not defined!")
		return
	}

	if len(cfg.RedmineApiKey) == 0 {
		log.Critical("REDMINE_API_KEY is not defined!")
		return
	}

	proxyController := controller.InitProxyController(cfg.RedmineApiKey, cfg.RedmineAddress)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router := mux.NewRouter()
	server := &http.Server{
		Addr:    cfg.HostPort,
		Handler: router,
	}
	serverErr := make(chan error, 1)

	go func() {

		router.HandleFunc("/", serveStatic).Methods("GET")
		router.HandleFunc("/api/", proxyController.ProxyRedmineAPI)
		router.HandleFunc("/health", healthCheck)

		log.Info("Try to start server...")
		err := server.ListenAndServe()
		if err != nil {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		server.Shutdown(ctx)
		log.Info("Escape signal received! Gentel shutdown!")
	case err := <-serverErr:
		log.Critical("Server can't run: %v", err)
	case <-ctx.Done():
		log.Error("Context DONE!!! Ouu my... this is terrible!")
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

// healthCheck проверка здоровья сервера
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"service":   "redmine-proxy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
