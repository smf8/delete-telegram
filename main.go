package main

import (
	"github.com/sirupsen/logrus"
	"sync"
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"net/http"
	"context"
	"time"
)

func main() {
	badgerDB, err := initDB(defaultConfig)
	if err != nil {
		logrus.Fatalf("failed to initialize database: %s", err.Error())
	}

	store := &Store{db: badgerDB}
	server := NewEchoServer(defaultConfig)
	authHandler := &AuthHandler{
		userStateMap: &sync.Map{},
		store:        store,
		config:       defaultConfig,
	}
	deleteHandler := &DeleteHandler{
		store:  store,
		config: defaultConfig,
	}

	RegisterEndpoints(server, authHandler, deleteHandler)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.Start(fmt.Sprintf(":%d", defaultConfig.ListenPort)); err != http.ErrServerClosed && err != nil {
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
