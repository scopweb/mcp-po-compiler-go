package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/scopweb/mcp-po-compiler-go/internal/mcp"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv := mcp.NewServer()
	if err := srv.Serve(ctx); err != nil {
		if err == context.Canceled {
			log.Println("shutdown requested")
			return
		}
		log.Printf("server stopped with error: %v\n", err)
		os.Exit(1)
	}
}
