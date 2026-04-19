package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Server represents a managed Ollama server process.
// If Ollama was already running when EnsureServer was called, Stop is a no-op.
type Server struct {
	cmd     *exec.Cmd
	managed bool // true only if this process started the server
}

// EnsureServer ensures an Ollama server is reachable at baseURL.
// If one is already running it returns immediately (managed=false).
// Otherwise it starts `ollama serve`, waits up to 30 s for it to be ready,
// and returns a Server whose Stop method will terminate the process.
func EnsureServer(baseURL string) (*Server, error) {
	healthURL := strings.TrimRight(baseURL, "/") + "/"

	if ping(healthURL) {
		log.Println("ollama: server already running, reusing")
		return &Server{managed: false}, nil
	}

	log.Println("ollama: server not running — starting `ollama serve`")
	cmd := exec.Command("ollama", "serve")
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ollama serve: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		if ping(healthURL) {
			log.Println("ollama: server ready")
			return &Server{cmd: cmd, managed: true}, nil
		}
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			return nil, fmt.Errorf("ollama server did not become ready within 30 seconds")
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// Stop terminates the server process if this process started it.
func (s *Server) Stop() {
	if !s.managed || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	log.Println("ollama: stopping server")
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		s.cmd.Process.Kill()
	}
	s.cmd.Wait()
}

// EnsureModel checks that model is available locally. Returns a clear error
// with instructions if it is not, so the server fails fast instead of letting
// every API call fail silently.
func EnsureModel(baseURL, model string) error {
	modelsURL := strings.TrimRight(baseURL, "/") + "/v1/models"
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get(modelsURL)
	if err != nil {
		return fmt.Errorf("could not reach Ollama to check models: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("could not parse model list: %w", err)
	}

	for _, m := range result.Data {
		if m.ID == model {
			return nil
		}
	}
	return fmt.Errorf("model %q is not pulled — run: make install-model", model)
}

func ping(url string) bool {
	c := &http.Client{Timeout: time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
