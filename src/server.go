package main

import (
	"html/template"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

type server struct {
	config *appConfig
	mux    *http.ServeMux
	tmpl   *template.Template
}

// TemplateData holds all the data needed for the HTML template
type TemplateData struct {
	ApplicationName      string
	ApplicationVersion   string
	ApplicationMessage   string
	Color                string
	Alive                bool
	Ready                bool
	RootDelaySeconds     int
	StartUpDelaySeconds  int
	TearDownDelaySeconds int
	LogToFileOnly        bool
	ProcessID            int
	UserID               int
	Hostname             string
	CatImageURL          string
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.ApplicationName}} {{.ApplicationVersion}}</title>
</head>
<body style='background-color:{{.Color}};'>
    <h1>{{.ApplicationName}}</h1>
    
    <h2>Configuration</h2>
    Application Version: {{.ApplicationVersion}}<br>
    Application Message: {{.ApplicationMessage}}<br>
    Application Liveness: {{.Alive}}<br>
    Application Readiness: {{.Ready}}<br>
    Delay seconds of root endpoint ('/'): {{.RootDelaySeconds}}<br>
    Seconds the application needs to start up: {{.StartUpDelaySeconds}}<br>
    Seconds the application needs to shut down gracefully: {{.TearDownDelaySeconds}}<br>
    Only log to file: {{.LogToFileOnly}}<br>
    
    <h2>Tech Details</h2>
    Process Id of the application: {{.ProcessID}}<br>
    User Id the application is using: {{.UserID}}<br>
    Hostname: {{.Hostname}}<br>
    
    {{if .CatImageURL}}
    <h2>The cute cat</h2>
    <img src='{{.CatImageURL}}' width='500px'></img>
    {{end}}
</body>
</html>`

func newServer(appConfig *appConfig) *server {
	// Parse the HTML template
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	// Create a new ServeMux
	mux := http.NewServeMux()

	server := &server{
		config: appConfig,
		mux:    mux,
		tmpl:   tmpl,
	}

	// Register handlers with the mux
	mux.HandleFunc("/", server.handleRoot)
	mux.HandleFunc("/favicon.ico", server.handleFavicon)
	mux.HandleFunc("/liveness", server.handleLiveness)
	mux.HandleFunc("/readiness", server.handleReadiness)

	return server
}

func (s *server) run() {
	err := http.ListenAndServe(":8080", s.mux)
	if err != nil {
		log.Errorf("Error on starting the server: '%s'", err)
	}
}

func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	log.Info("Request to root endpoint ('/')")

	if !s.config.rootEnabled {
		w.WriteHeader(http.StatusServiceUnavailable)
		log.Info("Root endpoint ('/') responded with Status Code 503 Service Unavailable due to root endpoint is disabled")
		return
	}

	// Handle delay if configured
	if s.config.rootDelaySeconds > 0 {
		for i := 0; i < s.config.rootDelaySeconds; i++ {
			log.Infof("Delayed Response for %d of %d seconds", i+1, s.config.rootDelaySeconds)
			time.Sleep(1 * time.Second)
		}
		log.Info("Finished delaying Response")
	}

	// Get hostname
	hostname, _ := os.Hostname()

	// Prepare template data
	data := TemplateData{
		ApplicationName:      s.config.applicationName,
		ApplicationVersion:   s.config.applicationVersion,
		ApplicationMessage:   s.config.applicationMessage,
		Color:                s.config.color,
		Alive:                s.config.alive,
		Ready:                s.config.ready,
		RootDelaySeconds:     s.config.rootDelaySeconds,
		StartUpDelaySeconds:  s.config.startUpDelaySeconds,
		TearDownDelaySeconds: s.config.tearDownDelaySeconds,
		LogToFileOnly:        s.config.logToFileOnly,
		ProcessID:            os.Getpid(),
		UserID:               os.Getuid(),
		Hostname:             hostname,
		CatImageURL:          s.config.catImageUrl,
	}

	// Set content type and execute template
	w.Header().Set("Content-Type", "text/html")
	if err := s.tmpl.Execute(w, data); err != nil {
		log.Errorf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (s *server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	log.Info("Request to liveness endpoint ('/liveness')")

	if s.config.alive {
		w.WriteHeader(http.StatusOK)
		log.Info("Liveness endpoint ('/liveness') responded with Status Code 200 OK")
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		log.Info("Liveness endpoint ('/liveness') responded with Status Code 500 Internal Server Error")
	}
}

func (s *server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	log.Info("Request to readiness endpoint ('/readiness')")

	if s.config.ready {
		w.WriteHeader(http.StatusOK)
		log.Info("Readiness endpoint ('/readiness') responded with Status Code 200 OK")
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		log.Info("Readiness endpoint ('/readiness') responded with Status Code 503 Service Unavailable")
	}
}
