package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kwkoo/configparser"
	"github.com/kwkoo/kserve-sd-frontend/internal"
)

//go:embed docroot/*
var content embed.FS

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func main() {
	config := struct {
		Port     int    `default:"8080" usage:"HTTP listener port"`
		Docroot  string `usage:"HTML document root - will use the embedded docroot if not specified"`
		ModelURL string `default:"http://localhost:8085/v2/models/sd/infer" usage:"Model URL"`
	}{}
	if err := configparser.Parse(&config); err != nil {
		log.Fatal(err)
	}

	var filesystem http.FileSystem
	if len(config.Docroot) > 0 {
		log.Printf("using %s in the file system as the document root", config.Docroot)
		filesystem = http.Dir(config.Docroot)
	} else {
		log.Print("using the embedded filesystem as the docroot")

		subdir, err := fs.Sub(content, "docroot")
		if err != nil {
			log.Fatalf("could not get subdirectory: %v", err)
		}
		filesystem = http.FS(subdir)
	}

	// Setup signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup

	fileServer := http.FileServer(filesystem).ServeHTTP
	http.HandleFunc("/healthz", health)
	http.HandleFunc("/", fileServer)

	modelClient := internal.NewModelClient(config.ModelURL)
	http.HandleFunc("/api/infer", func(w http.ResponseWriter, r *http.Request) {
		modelClient.Infer(ctx, w, r)
	})

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", config.Port),
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("listening on port %v", config.Port)
		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				log.Print("web server graceful shutdown")
				return
			}
			log.Fatal(err)
		}
	}()

	// Wait for SIGINT
	<-ctx.Done()
	stop()
	log.Print("interrupt signal received, shutting down web server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	wg.Wait()
	log.Print("shutdown successful")
}
