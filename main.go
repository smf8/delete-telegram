package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg := InitConfig()
	badgerDB, err := initDB(cfg)
	if err != nil {
		logrus.Fatalf("failed to initialize database: %s", err.Error())
	}

	store := &Store{db: badgerDB}
	server := NewEchoServer(cfg)
	authHandler := &AuthHandler{
		userStateMap: &sync.Map{},
		store:        store,
		config:       cfg,
	}
	deleteHandler := &DeleteHandler{
		store:  store,
		config: cfg,
	}

	RegisterEndpoints(server, authHandler, deleteHandler)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.Start(fmt.Sprintf(":%d", cfg.ListenPort)); err != http.ErrServerClosed && err != nil {
			server.Logger.Fatal(err.Error())
		}
	}()

	s := <-sig
	logrus.Infof("signal %s received\n", s)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Error(err.Error())
	}

}
