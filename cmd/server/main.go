package main

import (
	"fmt"
	"log"
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

	log.Printf("auto_gardener listening on %s", sanitizeStartupLogValue(listenURL(addr)))
	log.Printf("data dir: %s", sanitizeStartupLogValue(dataDir))
	log.Printf("static dir: %s", sanitizeStartupLogValue(staticDir))
	log.Printf("codex command: %s", sanitizeStartupLogValue(getenv("AUTO_GARDENER_CODEX_CMD", "codex")))
	log.Printf("claude command: %s", sanitizeStartupLogValue(getenv("AUTO_GARDENER_CLAUDE_CMD", "claude")))
	log.Printf("compat proxy: %s", sanitizeStartupLogValue(proxy.BaseURL()))
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

func sanitizeStartupLogValue(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\x%02x`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
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
