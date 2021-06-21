package tracker

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gtpaulose/uphold/internal/config"
	"github.com/gtpaulose/uphold/internal/db"
	cmap "github.com/orcaman/concurrent-map"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

type (
	// Tracker is the main tracking structure which is used to track the currency pairs defined
	Tracker struct {
		CurrencyPairs []string
		Settings

		// prices is a map of the currency pair to the tracked rate, map<string,Rate>
		// it tracks the bid and ask price seperately and does not correspond to the latest market values
		// for market rates, check console or the database
		// ConcurrentMap is used to prevent race conditions when accessing the various currency pairs as they happen simultaneously
		prices cmap.ConcurrentMap
	}

	// Settings is used to configure the tracker
	Settings struct {
		FetchInterval time.Duration
		OscPercentage float64
		Price         Price
	}

	// Price is enum of trackable price types
	Price string

	// Rate is the response received by the ticker endpoint
	Rate struct {
		Ask string
		Bid string
	}
)

const (
	Ask  Price = "ask"
	Bid  Price = "bid"
	Both Price = "both"
)

func (r Rate) GetAskPrice() float64 { return cast.ToFloat64(r.Ask) }

func (r Rate) GetBidPrice() float64 { return cast.ToFloat64(r.Bid) }

func InitTracker(cp []string, fi time.Duration, op float64, price Price) *Tracker {
	return &Tracker{
		CurrencyPairs: cp,
		Settings:      Settings{fi, op, price},
		prices:        cmap.New(),
	}
}

// Start will spawn routines for each currency pair defined in the tracker
// according to the fetch_interval, the prices are updated.
func (t *Tracker) Start(ctx context.Context) {
	log.Infoln("Starting tracker with following settings: ", t.Settings)

	for _, pair := range t.CurrencyPairs {
		// storing in another variable so it can safely passed to the goroutine
		cp := pair
		go func(ctx context.Context, db db.Storage) {
			log.Infoln("Starting tracker for currency pair: ", cp)

			ticker := time.NewTicker(t.FetchInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := t.trackCurrencyPair(cp, db); err != nil {
						fmt.Printf("Error in %s tracker: %s", cp, err.Error())
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(ctx, db.NewDB())
	}
}

// Stop will safely kill the spawned routines
func (t *Tracker) Stop(cancel context.CancelFunc) { cancel() }

func (t *Tracker) trackCurrencyPair(cp string, db db.Storage) error {
	var rate Rate

	if _, err := resty.New().R().
		EnableTrace().
		SetResult(&rate).
		Get(config.BaseURL + cp); err != nil {
		return err
	}

	log.Infof("Received rate for %s: %s\n", cp, rate)

	return t.update(cp, rate, db)
}

// getTrackedRate will receive the tracked rate for a currency pair
// for a given currency pair, the rate will be tracked seperately for both the bid and ask price
// the value in this map does not correspond to the latest market rate for the corresponding currency pair
func (t *Tracker) getTrackedRate(cp string) Rate {
	r, _ := t.prices.Get(cp)
	return r.(Rate)
}

func (t *Tracker) update(cp string, rate Rate, db db.Storage) error {
	// this will usually be executed after receiving the first response from the ticker endpoint
	if t.prices.SetIfAbsent(cp, rate) {
		return db.Store(NewRecord(cp, rate, 0, 0, beautifySettings(t.Settings, "")))
	}

	trackedRate := t.getTrackedRate(cp)

	if t.Price == Ask || t.Price == Both {
		diff := rate.GetAskPrice() - trackedRate.GetAskPrice()
		if math.Abs(diff) >= t.OscPercentage*trackedRate.GetAskPrice()/100 {
			log.Infof("ASK RATE FOR %s HAS CHANGED. NEW RATE: %v", cp, rate)

			trackedRate.Ask = rate.Ask
			t.prices.Set(cp, trackedRate)

			rec := NewRecord(cp, rate, diff, calculateDiffPercentage(diff, trackedRate.GetAskPrice()), beautifySettings(t.Settings, Ask))
			if err := db.Store(rec); err != nil {
				return err
			}
		}
	}

	if t.Price == Bid || t.Price == Both {
		diff := rate.GetBidPrice() - trackedRate.GetBidPrice()
		if math.Abs(diff) >= t.OscPercentage*trackedRate.GetBidPrice()/100 {
			log.Infof("BID RATE FOR %s HAS CHANGED. NEW RATE: %v", cp, rate)

			trackedRate.Bid = rate.Bid
			t.prices.Set(cp, trackedRate)

			rec := NewRecord(cp, rate, diff, calculateDiffPercentage(diff, trackedRate.GetBidPrice()), beautifySettings(t.Settings, Bid))
			if err := db.Store(rec); err != nil {
				return err
			}
		}
	}

	return nil
}

// beautifySettings will make sure the settings field will be readable in the database
func beautifySettings(settings Settings, price Price) Settings {
	settings.FetchInterval /= time.Second
	settings.Price = price

	return settings
}
