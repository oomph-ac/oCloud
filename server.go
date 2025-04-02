package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog"
)

var (
	logger zerolog.Logger
)

func init() {
	f, err := os.OpenFile("server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Unable to open log file: %v\n", err)
		os.Exit(1)
	}
	logger = zerolog.New(f)

	if sentryDsn := os.Getenv("SENTRY_DSN"); sentryDsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: sentryDsn,
		}); err != nil {
			fmt.Printf("Failed to initalize Sentry: %v\n", err)
			os.Exit(1)
		}
	}
}

func generateTLSConfig(pemFile, certFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, pemFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		InsecureSkipVerify: false,
		Certificates:       []tls.Certificate{cert},
		ServerName:         "*.oomph.ac",
	}, nil
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: ./oCloud <listen_addr> <pem_file> <cert_file>")
		return
	}

	listenAddr, pemFile, certFile := os.Args[1], os.Args[2], os.Args[3]
	fmt.Printf("Listening on %s with PEM file %s and cert file %s\n", listenAddr, pemFile, certFile)

	var interruptSignal = make(chan os.Signal, 1)
	signal.Notify(interruptSignal, os.Interrupt)

	tlsCfg, err := generateTLSConfig(pemFile, certFile)
	if err != nil {
		fmt.Printf("Failed to generate TLS config: %v\n", err)
		return
	}

	l, err := quic.ListenAddr(listenAddr, tlsCfg, &quic.Config{
		KeepAlivePeriod: time.Second,
		EnableDatagrams: false,
	})
	if err != nil {
		fmt.Printf("Failed to listen on %s: %v\n", listenAddr, err)
		return
	}

	go listen(l)
	<-interruptSignal
}
