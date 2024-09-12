package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/anoideaopen/channel-transfer/test/mock/batcher/grpc/app"
)

func main() {
	const defaultPort = 8888

	var (
		port   int
		logger *slog.Logger
	)

	flag.IntVar(&port, "port", defaultPort, "server port")
	flag.Parse()

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	application := app.New(logger, port)

	go application.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	application.Stop()
}
