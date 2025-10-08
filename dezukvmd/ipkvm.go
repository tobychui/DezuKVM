package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gorilla/csrf"
	"imuslab.com/dezukvm/dezukvmd/mod/dezukvm"
	"imuslab.com/dezukvm/dezukvmd/mod/utils"
)

var (
	dezukvmManager     *dezukvm.DezukVM
	csrfMiddleware     func(http.Handler) http.Handler
	listeningServerMux *http.ServeMux
)

func init() {
	csrfMiddleware = csrf.Protect(
		[]byte(nodeUUID),
		csrf.CookieName("dezukvm_csrf_token"),
		csrf.Secure(false),
		csrf.Path("/"),
		csrf.SameSite(csrf.SameSiteLaxMode),
	)
}

func init_ipkvm_mode() error {
	listeningServerMux = http.NewServeMux()
	//Create a new DezukVM manager
	dezukvmManager = dezukvm.NewKvmHostInstance(&dezukvm.RuntimeOptions{
		EnableLog: true,
	})

	// Experimental
	connectedUsbKvms, err := dezukvm.ScanConnectedUsbKvmDevices()
	if err != nil {
		return err
	}

	for _, dev := range connectedUsbKvms {
		err := dezukvmManager.AddUsbKvmDevice(dev)
		if err != nil {
			return err
		}
	}

	err = dezukvmManager.StartAllUsbKvmDevices()
	if err != nil {
		return err
	}
	// ~Experimental

	// Handle program exit to close the HID controller
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutting down DezuKVM...")

		if dezukvmManager != nil {
			dezukvmManager.Close()
		}
		log.Println("Shutdown complete.")
		os.Exit(0)
	}()

	// Middleware to inject CSRF token into HTML files served from www
	listeningServerMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only inject for .html files
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		if strings.HasSuffix(path, ".html") {
			// Read the HTML file from disk
			targetFilePath := filepath.Join("www", filepath.Clean(path))
			content, err := os.ReadFile(targetFilePath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			htmlContent := string(content)
			// Replace CSRF token placeholder
			htmlContent = strings.ReplaceAll(htmlContent, "{{.csrfToken}}", csrf.Token(r))
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(htmlContent))
			return
		}
		// Fallback to static file server for non-HTML files
		http.FileServer(http.Dir("www")).ServeHTTP(w, r)
	})

	// Register DezukVM related APIs
	register_ipkvm_apis(listeningServerMux)

	err = http.ListenAndServe(":9000", listeningServerMux)
	return err
}

func register_ipkvm_apis(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/stream/{uuid}/video", func(w http.ResponseWriter, r *http.Request) {
		instanceUUID := r.PathValue("uuid")
		fmt.Println("Requested video stream for instance UUID:", instanceUUID)
		dezukvmManager.HandleVideoStreams(w, r, instanceUUID)
	})

	mux.HandleFunc("/api/v1/stream/{uuid}/audio", func(w http.ResponseWriter, r *http.Request) {
		instanceUUID := r.PathValue("uuid")
		dezukvmManager.HandleAudioStreams(w, r, instanceUUID)
	})

	mux.HandleFunc("/api/v1/hid/{uuid}/events", func(w http.ResponseWriter, r *http.Request) {
		instanceUUID := r.PathValue("uuid")
		dezukvmManager.HandleHIDEvents(w, r, instanceUUID)
	})

	mux.HandleFunc("/api/v1/mass_storage/switch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		instanceUUID, err := utils.PostPara(r, "uuid")
		if err != nil {
			http.Error(w, "Missing or invalid uuid parameter", http.StatusBadRequest)
			return
		}
		side, err := utils.PostPara(r, "side")
		if err != nil {
			http.Error(w, "Missing or invalid side parameter", http.StatusBadRequest)
			return
		}
		switch side {
		case "kvm":
			dezukvmManager.HandleMassStorageSideSwitch(w, r, instanceUUID, true)
		case "remote":
			dezukvmManager.HandleMassStorageSideSwitch(w, r, instanceUUID, false)
		default:
			http.Error(w, "Invalid side parameter", http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/api/v1/instances", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			dezukvmManager.HandleListInstances(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
