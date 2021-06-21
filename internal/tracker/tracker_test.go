package tracker

import (
	"testing"
	"time"

	"github.com/gtpaulose/uphold/internal/db"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/stretchr/testify/assert"
)

const btc_usd string = "BTC-USD"
const eth_usd string = "ETH-USD"

func initMockTracker(price Price) *Tracker {
	return &Tracker{
		Settings:  getMockSettings(price),
		responses: cmap.New(),
	}
}

func getMockSettings(price Price) Settings {
	return Settings{1 * time.Second, 0.01, price}
}

func getDefaultRate() Rate { return Rate{"100", "100"} }

func TestTrackNewCurrPair(t *testing.T) {
	tracker := initMockTracker(Ask)
	tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))

	assert.Equal(t, tracker.responses.Count(), 1)
	rate, _ := tracker.responses.Get(btc_usd)
	assert.Equal(t, rate, getDefaultRate())
}

func TestUpdateCurrPair(t *testing.T) {
	tracker := initMockTracker(Ask)
	tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))
	tracker.update(btc_usd, Rate{"102", "100"}, db.NewMockDB(false))

	assert.Equal(t, tracker.responses.Count(), 1)
	rate, _ := tracker.responses.Get(btc_usd)
	assert.Equal(t, rate, Rate{"102", "100"})

	tracker.update(btc_usd, Rate{"103", "100"}, db.NewMockDB(false))
	assert.Equal(t, tracker.responses.Count(), 1)
	rate, _ = tracker.responses.Get(btc_usd)
	assert.Equal(t, rate, Rate{"102", "100"})

	tracker.update(btc_usd, Rate{"99", "100"}, db.NewMockDB(false))
	assert.Equal(t, tracker.responses.Count(), 1)
	rate, _ = tracker.responses.Get(btc_usd)
	assert.Equal(t, rate, Rate{"99", "100"})
}

func TestUpdateCurrPairBothPrices(t *testing.T) {
	tracker := initMockTracker(Both)
	tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))

	tracker.update(btc_usd, Rate{"99", "105"}, db.NewMockDB(false))
	assert.Equal(t, tracker.responses.Count(), 1)
	rate, _ := tracker.responses.Get(btc_usd)
	assert.Equal(t, rate, Rate{"99", "105"})

	tracker.update(btc_usd, Rate{"99.5", "103"}, db.NewMockDB(false))
	assert.Equal(t, tracker.responses.Count(), 1)
	rate, _ = tracker.responses.Get(btc_usd)
	assert.Equal(t, rate, Rate{"99", "103"})

	tracker.update(btc_usd, Rate{"102", "102"}, db.NewMockDB(false))
	assert.Equal(t, tracker.responses.Count(), 1)
	rate, _ = tracker.responses.Get(btc_usd)
	assert.Equal(t, rate, Rate{"102", "103"})
}

func TestConcurrentUpdateCurrPair(t *testing.T) {
	tracker := initMockTracker(Both)

	go func() {
		tracker.update(eth_usd, getDefaultRate(), db.NewMockDB(false))
	}()

	time.Sleep(100 * time.Millisecond)
	tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))

	assert.Equal(t, tracker.responses.Count(), 2)
	rate, _ := tracker.responses.Get(btc_usd)
	assert.Equal(t, rate, getDefaultRate())
	rate, _ = tracker.responses.Get(eth_usd)
	assert.Equal(t, rate, getDefaultRate())
}
