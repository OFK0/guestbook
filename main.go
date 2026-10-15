// Command guestbook is a tiny web app to try piway with: sign the book,
// see the latest entries, and watch a deploy replace the container.
//
// Entries live in memory, so a deploy starts a fresh book: piway gives
// containers no persistent storage yet.
package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"
)

//go:embed page.html
var files embed.FS

var page = template.Must(template.ParseFS(files, "page.html"))

// version is set at build time: -ldflags "-X main.version=..."
var version = "dev"

const (
	maxEntries = 100
	maxName    = 40
	maxMessage = 280
)

type entry struct {
	Name, Message string
	At            time.Time
}

type book struct {
	mu      sync.Mutex
	entries []entry // newest first
	visits  int
}

func (b *book) sign(name, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append([]entry{{name, message, time.Now()}}, b.entries...)
	if len(b.entries) > maxEntries {
		b.entries = b.entries[:maxEntries]
	}
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > n {
		s = string([]rune(s)[:n])
	}
	return s
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	host, _ := os.Hostname()
	started := time.Now()
	b := &book{}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		b.mu.Lock()
		b.visits++
		data := map[string]any{
			"Entries": append([]entry(nil), b.entries...), "Visits": b.visits,
			"Version": version, "Host": host, "Started": started, "Up": time.Since(started).Round(time.Second),
			"MaxName": maxName, "MaxMessage": maxMessage,
		}
		b.mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, data); err != nil {
			log.Printf("render: %v", err)
		}
	})
	mux.HandleFunc("POST /sign", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		name, message := clip(r.FormValue("name"), maxName), clip(r.FormValue("message"), maxMessage)
		if name == "" || message == "" {
			http.Error(w, "a name and a message, please", http.StatusBadRequest)
			return
		}
		b.sign(name, message)
		log.Printf("signed by %q", name)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	srv := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("guestbook %s listening on :%s", version, port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// piway stops the old container with SIGTERM after a deploy: finish the
	// requests in flight, then exit.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	<-ctx.Done()
	log.Print("shutting down")
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdown)
}
