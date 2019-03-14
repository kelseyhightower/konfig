package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/kelseyhightower/konfig"
)

func main() {
	log.Println("Starting env service ...")

	httpListenPort := os.Getenv("PORT")
	if httpListenPort == "" {
		httpListenPort = "8080"
	}

	hostPort := net.JoinHostPort("0.0.0.0", httpListenPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, e := range os.Environ() {
			pair := strings.Split(e, "=")
			w.Header().Set(pair[0], pair[1])
		}

		data, err := ioutil.ReadFile(os.Getenv("CONFIG_FILE"))
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "", 500)
			return
		}

		fmt.Fprintf(w, "  %s\n", data)
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