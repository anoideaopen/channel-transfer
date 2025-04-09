package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/anoideaopen/channel-transfer/pkg/app"
	"github.com/anoideaopen/channel-transfer/pkg/config"
)

var AppInfoVer = "undefined-ver"

//nolint:funlen
func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	if err = app.Run(ctx, cfg, AppInfoVer); err != nil {
		fmt.Printf("%+v\n", err)
		exitCode = 1
	}
}
