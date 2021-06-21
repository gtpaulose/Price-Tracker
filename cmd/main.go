package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/gtpaulose/uphold/internal/config"
	"github.com/gtpaulose/uphold/internal/tracker"
)

func main() {
	config.InitConfig()
	t := tracker.InitTracker(
		config.GetCurrencyPairs(),
		config.GetFetchInterval(),
		config.GetOscPercentage(),
		tracker.Price(config.GetPrice()),
	)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	t.Start(ctx)

	<-interrupt
	t.Stop(cancel)
}
