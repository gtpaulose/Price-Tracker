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
	Tracker struct {
		CurrencyPairs []string
		Settings

		responses cmap.ConcurrentMap
	}

	Settings struct {
		FetchInterval time.Duration
		OscPercentage float64
		Price         Price
	}

	Price string

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
		responses:     cmap.New(),
	}
}

func (t *Tracker) Start(ctx context.Context) {
	log.Infoln("Starting tracker with following settings: ", t.Settings)
	for _, currPair := range t.CurrencyPairs {
		cp := currPair
		go func(ctx context.Context, db db.Storage) {
			fmt.Println("Starting tracker for currency pair: ", cp)
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

func (t *Tracker) Stop(cancel context.CancelFunc) { cancel() }

func (t *Tracker) trackCurrencyPair(cp string, db db.Storage) error {
	var rate Rate

	if _, err := resty.New().R().
		EnableTrace().
		SetResult(&rate).
		Get(config.BaseURL + cp); err != nil {
		return err
	}

	fmt.Printf("Received rate for %s: %s\n", cp, rate)

	return t.update(cp, rate, db)
}

func (t *Tracker) getTrackedRate(cp string) Rate {
	r, _ := t.responses.Get(cp)
	return r.(Rate)
}

func (t *Tracker) update(cp string, rate Rate, db db.Storage) error {
	if t.responses.SetIfAbsent(cp, rate) {
		return db.Store(NewRecord(cp, rate, 0, 0, beautifySettings(t.Settings, "")))
	}

	trackedRate := t.getTrackedRate(cp)

	if t.Price == Ask || t.Price == Both {
		diff := rate.GetAskPrice() - trackedRate.GetAskPrice()
		if math.Abs(diff) >= t.OscPercentage*trackedRate.GetAskPrice()/100 {
			log.Infof("ASK RATE FOR %s HAS CHANGED. NEW RATE: %v", cp, rate)
			trackedRate.Ask = rate.Ask
			t.responses.Set(cp, trackedRate)

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
			t.responses.Set(cp, trackedRate)

			rec := NewRecord(cp, rate, diff, calculateDiffPercentage(diff, trackedRate.GetBidPrice()), beautifySettings(t.Settings, Bid))
			if err := db.Store(rec); err != nil {
				return err
			}
		}
	}

	return nil
}

func beautifySettings(settings Settings, price Price) Settings {
	settings.FetchInterval /= time.Second
	settings.Price = price

	return settings
}
