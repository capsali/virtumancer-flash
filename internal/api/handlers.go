package api

import (
	"encoding/json"
	"net/http"

	"github.com/capsali/virtumancer/internal/console"
	"github.com/capsali/virtumancer/internal/libvirt"
	"github.com/capsali/virtumancer/internal/services"
	"github.com/capsali/virtumancer/internal/storage"
	"github.com/capsali/virtumancer/internal/ws"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type APIHandler struct {
	HostService services.HostServiceProvider
	Hub         *ws.Hub
	DB          *gorm.DB
	Connector   *libvirt.Connector
}

func NewAPIHandler(hostService services.HostServiceProvider, hub *ws.Hub, db *gorm.DB, connector *libvirt.Connector) *APIHandler {
	return &APIHandler{
		HostService: hostService,
		Hub:         hub,
		DB:          db,
		Connector:   connector,
	}
}

func (h *APIHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws.ServeWs(h.Hub, h.HostService, w, r)
}

func (h *APIHandler) HandleVMConsole(w http.ResponseWriter, r *http.Request) {
	console.HandleConsole(h.DB, h.Connector, w, r)
}

func (h *APIHandler) HandleSpiceConsole(w http.ResponseWriter, r *http.Request) {
	console.HandleSpiceConsole(h.DB, h.Connector, w, r)
}

func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *APIHandler) CreateHost(w http.ResponseWriter, r *http.Request) {
	var host storage.Host
	if err := json.NewDecoder(r.Body).Decode(&host); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	newHost, err := h.HostService.AddHost(host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newHost)
}

func (h *APIHandler) GetHosts(w http.ResponseWriter, r *http.Request) {
	hosts, err := h.HostService.GetAllHosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hosts)
}

func (h *APIHandler) GetHostInfo(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	info, err := h.HostService.GetHostInfo(hostID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *APIHandler) DeleteHost(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	if err := h.HostService.RemoveHost(hostID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListVMsFromLibvirt gets the unified view of VMs for a host.
func (h *APIHandler) ListVMsFromLibvirt(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")

	// Immediately get VMs from the DB for a fast response.
	vms, err := h.HostService.GetVMsForHostFromDB(hostID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// In the background, trigger a sync from libvirt.
	// The service will broadcast a websocket update when it's done.
	go h.HostService.SyncVMsForHost(hostID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vms)
}

func (h *APIHandler) GetVMStats(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")
	stats, err := h.HostService.GetVMStats(hostID, vmName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *APIHandler) GetVMHardware(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")
	hardware, err := h.HostService.GetVMHardwareAndTriggerSync(hostID, vmName)
	if err != nil {
		// Even if there's an error (e.g., no cache yet), we might still proceed
		// if we want to allow the background sync to populate it.
		// For now, we'll return an error if the initial fetch fails.
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hardware)
}

// --- VM Actions ---

func (h *APIHandler) StartVM(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")
	if err := h.HostService.StartVM(hostID, vmName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) ShutdownVM(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")
	if err := h.HostService.ShutdownVM(hostID, vmName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) RebootVM(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")
	if err := h.HostService.RebootVM(hostID, vmName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) ForceOffVM(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")
	if err := h.HostService.ForceOffVM(hostID, vmName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) ForceResetVM(w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")
	if err := h.HostService.ForceResetVM(hostID, vmName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}


