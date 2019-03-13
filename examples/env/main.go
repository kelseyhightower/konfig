package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/cloudrun/kubernetes"
)

func main() {
	if err := kubernetes.Parse(); err != nil {
		log.Println(err)
	}

	log.Println("Starting env service ...")

	httpListenPort := os.Getenv("PORT")
	if httpListenPort == "" {
		httpListenPort = "8080"
	}

	hostPort := net.JoinHostPort("0.0.0.0", httpListenPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, e := range os.Environ() {
			fmt.Fprintf(w, "%s\n", e)
		}
	})

	s := &http.Server{
		Addr:    hostPort,
		Handler: mux,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	signalCh := make(chan os.Signal)
	signal.Notify(signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
	)

	<-signalCh
	log.Println("Shutdown called....")

	shutdownContext := context.Background()

	if err := s.Shutdown(shutdownContext); err != nil {
		log.Fatal(err)
	}
}
