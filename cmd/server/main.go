package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"auto_gardener/internal/app"
	"auto_gardener/internal/codex"
	"auto_gardener/internal/compat"
)

func main() {
	addr := getenv("AUTO_GARDENER_ADDR", "127.0.0.1:8080")
	dataDir := app.DefaultDataDir()
	staticDir := defaultStaticDir()

	events := app.NewEventHub()
	store, err := app.NewStore(dataDir, events)
	if err != nil {
		log.Fatal(err)
	}
	proxy, err := compat.Start()
	if err != nil {
		log.Fatal(err)
	}
	orch := app.NewOrchestrator(store, codex.NewRunnerFromEnv(), dataDir, proxy.BaseURL())
	orch.ResumeUnfinished()
	orch.StartWatchdog()
	server := app.NewServer(store, orch, staticDir, events)

	log.Printf("auto_gardener listening on %s", listenURL(addr))
	log.Printf("data dir: %s", dataDir)
	log.Printf("static dir: %s", staticDir)
	log.Printf("codex command: %s", getenv("AUTO_GARDENER_CODEX_CMD", "codex"))
	log.Printf("claude command: %s", getenv("AUTO_GARDENER_CLAUDE_CMD", "claude"))
	log.Printf("compat proxy: %s", proxy.BaseURL())
	if isExternalBind(addr) && os.Getenv("AUTO_GARDENER_ALLOW_EXTERNAL_BIND") != "1" {
		log.Fatalf("refusing to listen on non-loopback address %q without AUTO_GARDENER_ALLOW_EXTERNAL_BIND=1", addr)
	}
	if power := app.CheckPowerStatus(); !power.OK {
		log.Printf("power warning: remote access requires this computer to stay awake and powered on; %s", app.PowerWarningsText(power))
	}
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}

func listenURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	return "http://" + addr
}

func getenv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func defaultStaticDir() string {
	if v := strings.TrimSpace(os.Getenv("AUTO_GARDENER_STATIC")); v != "" {
		return v
	}
	candidates := []string{"web/static"}
	if exe, err := os.Executable(); err == nil && exe != "" {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "web", "static"),
			filepath.Join(exeDir, "..", "web", "static"),
		)
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return "web/static"
}

func isExternalBind(addr string) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return false
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			host = ""
		} else {
			host = addr
		}
	}
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return false
	}
	if host == "" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback()
	}
	return true
}
