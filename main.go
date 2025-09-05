package main

import (
	"log"
	"net/http"
	"os"

	"github.com/capsali/virtumancer/internal/api"
	"github.com/capsali/virtumancer/internal/libvirt"
	"github.com/capsali/virtumancer/internal/services"
	"github.com/capsali/virtumancer/internal/storage"
	"github.com/capsali/virtumancer/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Initialize Database
	db, err := storage.InitDB("virtumancer.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// Initialize Libvirt Connector
	connector := libvirt.NewConnector()

	// Initialize Host Service, now with WebSocket hub
	hostService := services.NewHostService(db, connector, hub)

	// On startup, load all hosts from DB and try to connect
	hostService.ConnectToAllHosts()

	// Initialize API Handler, now with WebSocket hub
	apiHandler := api.NewAPIHandler(hostService, hub)

	// Setup Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", apiHandler.HealthCheck)

		// Host routes
		r.Get("/hosts", apiHandler.GetHosts)
		r.Post("/hosts", apiHandler.CreateHost)
		r.Delete("/hosts/{hostID}", apiHandler.DeleteHost)

		// VM routes
		r.Get("/hosts/{hostID}/vms", apiHandler.ListVMs)
		r.Post("/hosts/{hostID}/vms/{vmName}/start", apiHandler.StartVM)
		r.Post("/hosts/{hostID}/vms/{vmName}/shutdown", apiHandler.ShutdownVM)
		r.Post("/hosts/{hostID}/vms/{vmName}/reboot", apiHandler.RebootVM)
		r.Post("/hosts/{hostID}/vms/{vmName}/forceoff", apiHandler.ForceOffVM)
		r.Post("/hosts/{hostID}/vms/{vmName}/forcereset", apiHandler.ForceResetVM)
	})

	// WebSocket route
	r.HandleFunc("/ws", apiHandler.HandleWebSocket)

	// Static File Server for the Vue App
	workDir, _ := os.Getwd()
	fileServer := http.FileServer(http.Dir(workDir + "/web/dist"))
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		_, err := os.Stat(workDir + "/web/dist" + r.URL.Path)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, workDir+"/web/dist/index.html")
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	log.Println("Starting server on :8888")
	err = http.ListenAndServe(":8888", r)
	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}


