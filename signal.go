package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func mainContext() context.Context {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	ctx, stop := context.WithCancel(context.Background())
	go func() {
		sig := <-c
		log.Printf("INFO: Received signal %v", sig)
		stop()
		signal.Stop(c)
	}()
	return ctx
}
