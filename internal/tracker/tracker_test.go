package tracker

import (
	"testing"
	"time"

	"github.com/gtpaulose/uphold/internal/db"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/stretchr/testify/assert"
)

// Test cases track the price changes over a 1% oscillation window, to improve code readability and understanding

const btc_usd string = "BTC-USD"
const eth_usd string = "ETH-USD"

func initMockTracker(price Price) *Tracker {
	return &Tracker{
		Settings: getMockSettings(price),
		prices:   cmap.New(),
	}
}

func getMockSettings(price Price) Settings {
	return Settings{1 * time.Second, 1, price}
}

func getDefaultRate() Rate { return Rate{"100", "100"} }

func TestTrackNewCurrPair(t *testing.T) {
	tracker := initMockTracker(Ask)
	err := tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))
	assert.NoError(t, err)

	assert.Equal(t, tracker.prices.Count(), 1)
	rate, _ := tracker.prices.Get(btc_usd)
	assert.Equal(t, rate, getDefaultRate())
}

func TestErrorDB(t *testing.T) {
	tracker := initMockTracker(Ask)
	err := tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))
	assert.NoError(t, err)

	err = tracker.update(btc_usd, Rate{"102", "100"}, db.NewMockDB(true))
	assert.Error(t, err)

	tracker = initMockTracker(Bid)
	err = tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))
	assert.NoError(t, err)

	err = tracker.update(btc_usd, Rate{"100", "102"}, db.NewMockDB(true))
	assert.Error(t, err)
}

func TestUpdateCurrPair(t *testing.T) {
	tracker := initMockTracker(Ask)
	err := tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))
	assert.NoError(t, err)

	// ask price changes above threshold -> update
	err = tracker.update(btc_usd, Rate{"102", "100"}, db.NewMockDB(false))
	assert.NoError(t, err)
	rate, _ := tracker.prices.Get(btc_usd)
	assert.Equal(t, rate, Rate{"102", "100"})

	// ask price changes below threshold -> no update
	err = tracker.update(btc_usd, Rate{"103", "100"}, db.NewMockDB(false))
	assert.NoError(t, err)
	rate, _ = tracker.prices.Get(btc_usd)
	assert.Equal(t, rate, Rate{"102", "100"})

	// ask price changes above threshold -> update
	err = tracker.update(btc_usd, Rate{"99", "100"}, db.NewMockDB(false))
	assert.NoError(t, err)
	rate, _ = tracker.prices.Get(btc_usd)
	assert.Equal(t, rate, Rate{"99", "100"})
}

func TestUpdateCurrPairBothPrices(t *testing.T) {
	tracker := initMockTracker(Both)
	err := tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))
	assert.NoError(t, err)

	// both ask and big price change above threshold -> update
	err = tracker.update(btc_usd, Rate{"99", "105"}, db.NewMockDB(false))
	assert.NoError(t, err)
	rate, _ := tracker.prices.Get(btc_usd)
	assert.Equal(t, rate, Rate{"99", "105"})

	// ask price changes below threshold -> no update
	err = tracker.update(btc_usd, Rate{"99.5", "103"}, db.NewMockDB(false))
	assert.NoError(t, err)
	rate, _ = tracker.prices.Get(btc_usd)
	assert.Equal(t, rate, Rate{"99", "103"})

	// bid price changes below threshold -> no update
	err = tracker.update(btc_usd, Rate{"102", "102"}, db.NewMockDB(false))
	assert.NoError(t, err)
	rate, _ = tracker.prices.Get(btc_usd)
	assert.Equal(t, rate, Rate{"102", "103"})
}

func TestConcurrentUpdateCurrPair(t *testing.T) {
	tracker := initMockTracker(Both)

	go func() {
		err := tracker.update(eth_usd, getDefaultRate(), db.NewMockDB(false))
		assert.NoError(t, err)
	}()
	err := tracker.update(btc_usd, getDefaultRate(), db.NewMockDB(false))
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, tracker.prices.Count(), 2)
	rate, _ := tracker.prices.Get(btc_usd)
	assert.Equal(t, rate, getDefaultRate())
	rate, _ = tracker.prices.Get(eth_usd)
	assert.Equal(t, rate, getDefaultRate())
}
