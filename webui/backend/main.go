package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gick-web-ui/frontend"
	"github.com/goccy/go-yaml"
)

type historyEntry struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  string    `json:"duration"`
	ExitCode  int       `json:"exit_code"`
	Stopped   bool      `json:"stopped"`
	Error     string    `json:"error"`
}

type processState struct {
	Running       bool      `json:"running"`
	PID           int       `json:"pid"`
	StartTime     time.Time `json:"start_time"`
	StopRequested bool      `json:"stop_requested"`
	LastExitCode  int       `json:"last_exit_code"`
	LastError     string    `json:"last_error"`
}

type server struct {
	mu           sync.Mutex
	state        processState
	cmd          *exec.Cmd
	logLines     []string
	maxLogLines  int
	stateDir     string
	historyPath  string
	configPath   string
	gickupBin    string
	lastMetrics  time.Time
	metricsError string
}

type prometheusConf struct {
	ListenAddr string `yaml:"listen_addr"`
	Endpoint   string `yaml:"endpoint"`
}

type metricsConf struct {
	Prometheus prometheusConf `yaml:"prometheus"`
}

type confFile struct {
	Metrics metricsConf `yaml:"metrics"`
}

func main() {
	listenAddr := flag.String("listen", ":3780", "listen address for web UI")
	configPath := flag.String("conf", "conf.yml", "path to config file")
	gickupBin := flag.String("gickup-bin", "", "path to gickup binary (default resolves automatically)")
	stateDir := flag.String("state-dir", ".gickup-ui", "directory for web UI state")
	flag.Parse()

	resolvedBin := resolveGickupBinary(*gickupBin)

	srv := &server{
		maxLogLines: 2000,
		stateDir:    *stateDir,
		historyPath: filepath.Join(*stateDir, "history.json"),
		configPath:  *configPath,
		gickupBin:   resolvedBin,
	}

	if err := os.MkdirAll(*stateDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create state dir: %v\n", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", srv.handleStatus)
	mux.HandleFunc("/api/start", srv.handleStart)
	mux.HandleFunc("/api/stop", srv.handleStop)
	mux.HandleFunc("/api/config", srv.handleConfig)
	mux.HandleFunc("/api/config/download", srv.handleConfigDownload)
	mux.HandleFunc("/api/validate", srv.handleValidate)
	mux.HandleFunc("/api/history", srv.handleHistory)
	mux.HandleFunc("/api/history/clear", srv.handleHistoryClear)
	mux.HandleFunc("/api/logs", srv.handleLogs)
	mux.HandleFunc("/api/logs/clear", srv.handleLogsClear)
	mux.HandleFunc("/api/metrics", srv.handleMetrics)
	mux.Handle("/", srv.handleStatic())

	fmt.Printf("gickup-web listening on %s\n", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func resolveGickupBinary(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("GICKUP_BIN"); env != "" {
		return env
	}

	exe, err := os.Executable()
	if err == nil {
		binDir := filepath.Dir(exe)
		candidates := []string{"gickup"}
		if runtime.GOOS == "windows" {
			candidates = append(candidates, "gickup.exe")
		}
		for _, name := range candidates {
			candidate := filepath.Join(binDir, name)
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate
			}
		}
	}

	return "gickup"
}

func (s *server) handleStatic() http.Handler {
	sub, err := fs.Sub(webui.FS, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "web UI not built", http.StatusServiceUnavailable)
		})
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(sub, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback
		index, err := sub.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer index.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, index)
	})
}

func (s *server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	respondJSON(w, http.StatusOK, s.state)
}

func (s *server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data, err := os.ReadFile(s.configPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.Write(data)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(bytes.TrimSpace(body)) == 0 {
		http.Error(w, "empty config", http.StatusBadRequest)
		return
	}

	if err := os.WriteFile(s.configPath, body, 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *server) handleConfigDownload(w http.ResponseWriter, _ *http.Request) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=conf.yml")
	w.Write(data)
}

func (s *server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(bytes.TrimSpace(body)) == 0 {
		http.Error(w, "empty config", http.StatusBadRequest)
		return
	}

	validated, output := s.validateConfig(body)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":  validated,
		"output": output,
	})
}

func (s *server) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	if s.state.Running {
		state := s.state
		s.mu.Unlock()
		respondJSON(w, http.StatusConflict, state)
		return
	}
	cmd := exec.Command(s.gickupBin, s.configPath)
	cmd.Dir = filepath.Dir(s.configPath)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		s.state.LastError = err.Error()
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.state = processState{
		Running:       true,
		PID:           cmd.Process.Pid,
		StartTime:     time.Now(),
		StopRequested: false,
		LastExitCode:  0,
		LastError:     "",
	}
	s.cmd = cmd
	s.mu.Unlock()

	go s.captureLogs(stdout)
	go s.captureLogs(stderr)
	go s.waitForExit(cmd)

	s.appendHistoryStart()

	s.mu.Lock()
	state := s.state
	s.mu.Unlock()
	respondJSON(w, http.StatusOK, state)
}

func (s *server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	if !s.state.Running || s.cmd == nil || s.cmd.Process == nil {
		state := s.state
		s.mu.Unlock()
		respondJSON(w, http.StatusOK, state)
		return
	}

	s.state.StopRequested = true
	cmd := s.cmd
	s.mu.Unlock()

	_ = cmd.Process.Kill()

	respondJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
}

func (s *server) handleHistory(w http.ResponseWriter, _ *http.Request) {
	history := s.loadHistory()
	respondJSON(w, http.StatusOK, history)
}

func (s *server) handleHistoryClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.saveHistory([]historyEntry{})

	respondJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *server) handleLogs(w http.ResponseWriter, r *http.Request) {
	lines := 200
	if param := r.URL.Query().Get("lines"); param != "" {
		if parsed, err := fmt.Sscanf(param, "%d", &lines); err != nil || parsed != 1 {
			lines = 200
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if lines <= 0 || lines > len(s.logLines) {
		lines = len(s.logLines)
	}
	start := len(s.logLines) - lines
	if start < 0 {
		start = 0
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"lines": s.logLines[start:],
	})
}

func (s *server) handleLogsClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	s.logLines = []string{}
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	metricsURL, err := s.metricsURL()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := http.Get(metricsURL)
	if err != nil {
		s.mu.Lock()
		s.metricsError = err.Error()
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	s.mu.Lock()
	s.lastMetrics = time.Now()
	s.metricsError = ""
	s.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	w.Write(data)
}

func (s *server) validateConfig(content []byte) (bool, string) {
	file, err := os.CreateTemp("", "gickup-validate-*.yml")
	if err != nil {
		return false, err.Error()
	}
	defer os.Remove(file.Name())

	if _, err := file.Write(content); err != nil {
		return false, err.Error()
	}
	file.Close()

	cmd := exec.Command(s.gickupBin, "--dryrun", file.Name())
	cmd.Dir = filepath.Dir(s.configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, string(output)
	}

	return true, string(output)
}

func (s *server) captureLogs(reader io.ReadCloser) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		s.mu.Lock()
		s.logLines = append(s.logLines, line)
		if len(s.logLines) > s.maxLogLines {
			s.logLines = s.logLines[len(s.logLines)-s.maxLogLines:]
		}
		s.mu.Unlock()
	}
}

func (s *server) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()
	end := time.Now()

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	s.mu.Lock()
	start := s.state.StartTime
	stopped := s.state.StopRequested
	s.state.Running = false
	s.state.PID = 0
	s.state.LastExitCode = exitCode
	if err != nil {
		s.state.LastError = err.Error()
	} else {
		s.state.LastError = ""
	}
	s.cmd = nil
	s.mu.Unlock()

	s.appendHistoryComplete(start, end, exitCode, stopped, err)
}

func (s *server) appendHistoryStart() {
	s.mu.Lock()
	start := s.state.StartTime
	s.mu.Unlock()

	entries := s.loadHistory()
	entry := historyEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		StartTime: start,
		EndTime:   time.Time{},
		Duration:  "",
		ExitCode:  0,
		Stopped:   false,
		Error:     "",
	}
	entries = append([]historyEntry{entry}, entries...)
	s.saveHistory(entries)
}

func (s *server) appendHistoryComplete(start, end time.Time, exitCode int, stopped bool, err error) {
	entries := s.loadHistory()
	if len(entries) == 0 {
		entry := historyEntry{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			StartTime: start,
			EndTime:   end,
			Duration:  end.Sub(start).String(),
			ExitCode:  exitCode,
			Stopped:   stopped,
		}
		if err != nil {
			entry.Error = err.Error()
		}
		entries = append([]historyEntry{entry}, entries...)
		s.saveHistory(entries)
		return
	}

	entries[0].EndTime = end
	entries[0].Duration = end.Sub(start).String()
	entries[0].ExitCode = exitCode
	entries[0].Stopped = stopped
	if err != nil {
		entries[0].Error = err.Error()
	}
	// keep the list capped
	if len(entries) > 100 {
		entries = entries[:100]
	}
	s.saveHistory(entries)
}

func (s *server) loadHistory() []historyEntry {
	data, err := os.ReadFile(s.historyPath)
	if err != nil {
		return []historyEntry{}
	}
	var entries []historyEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []historyEntry{}
	}

	return entries
}

func (s *server) saveHistory(entries []historyEntry) {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.historyPath, data, 0o644)
}

func (s *server) metricsURL() (string, error) {
	conf, err := s.loadFirstConfig()
	if err != nil {
		return "", err
	}
	listen := conf.Metrics.Prometheus.ListenAddr
	endpoint := conf.Metrics.Prometheus.Endpoint
	if listen == "" || endpoint == "" {
		return "", fmt.Errorf("prometheus not configured")
	}

	if strings.HasPrefix(listen, ":") {
		listen = "127.0.0.1" + listen
	}
	if !strings.Contains(listen, ":") {
		listen = listen + ":6178"
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	return "http://" + listen + endpoint, nil
}

func (s *server) loadFirstConfig() (*confFile, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var c confFile
		decodeErr := dec.Decode(&c)
		if decodeErr == io.EOF {
			break
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		return &c, nil
	}

	return nil, fmt.Errorf("empty config")
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}
