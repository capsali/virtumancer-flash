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

	// Initialize Host Service
	hostService := services.NewHostService(db, connector, hub)

	// On startup, load all hosts from DB and try to connect
	hostService.ConnectToAllHosts()

	// Initialize API Handler
	apiHandler := api.NewAPIHandler(hostService, hub, db, connector)

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
		r.Get("/hosts/{hostID}/info", apiHandler.GetHostInfo)
		r.Delete("/hosts/{hostID}", apiHandler.DeleteHost)

		// VM routes
		r.Get("/hosts/{hostID}/vms", apiHandler.ListVMsFromLibvirt)
		r.Post("/hosts/{hostID}/vms/{vmName}/start", apiHandler.StartVM)
		r.Post("/hosts/{hostID}/vms/{vmName}/shutdown", apiHandler.ShutdownVM)
		r.Post("/hosts/{hostID}/vms/{vmName}/reboot", apiHandler.RebootVM)
		r.Post("/hosts/{hostID}/vms/{vmName}/forceoff", apiHandler.ForceOffVM)
		r.Post("/hosts/{hostID}/vms/{vmName}/forcereset", apiHandler.ForceResetVM)
		r.Get("/hosts/{hostID}/vms/{vmName}/stats", apiHandler.GetVMStats)
		r.Get("/hosts/{hostID}/vms/{vmName}/hardware", apiHandler.GetVMHardware)

		// Console routes
		r.Get("/hosts/{hostID}/vms/{vmName}/console", apiHandler.HandleVMConsole)
		r.Get("/hosts/{hostID}/vms/{vmName}/spice", apiHandler.HandleSpiceConsole)
	})

	// WebSocket route for UI updates
	r.HandleFunc("/ws", apiHandler.HandleWebSocket)

	// Static File Server for the Vue App
	workDir, _ := os.Getwd()

	spiceDir := http.Dir(workDir + "/web/public/spice")
	r.Handle("/spice/*", http.StripPrefix("/spice/", http.FileServer(spiceDir)))

	fileServer := http.FileServer(http.Dir(workDir + "/web/dist"))
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		_, err := os.Stat(workDir + "/web/dist" + r.URL.Path)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, workDir+"/web/dist/index.html")
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	certFile := "localhost.crt"
	keyFile := "localhost.key"

	log.Println("Starting HTTPS server on :8888")
	err = http.ListenAndServeTLS(":8888", certFile, keyFile, r)
	if err != nil {
		log.Printf("Could not start HTTPS server: %v", err)
		log.Println("Please ensure 'localhost.crt' and 'localhost.key' are present in the root directory.")
		log.Println("You can generate them by running the 'generate-certs.sh' script.")
	}
}


