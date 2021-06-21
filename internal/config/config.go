package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

const BaseURL string = "https://api-sandbox.uphold.com/v0/ticker/"

func InitConfig() {
	viper.AutomaticEnv()

	// Set default currency pair
	viper.SetDefault("CURRENCY_PAIRS", "BTC-USD")

	// Set default tracker settings
	viper.SetDefault("FETCH_INTERVAL", 5)
	viper.SetDefault("OSC_PERCENTAGE", 0.01)
	viper.SetDefault("PRICE", "BOTH")
}

func GetCurrencyPairs() []string { return strings.Split(viper.GetString("CURRENCY_PAIRS"), ",") }

func GetFetchInterval() time.Duration { return viper.GetDuration("FETCH_INTERVAL") * time.Second }

func GetOscPercentage() float64 { return viper.GetFloat64("OSC_PERCENTAGE") }

func GetPrice() string { return strings.ToLower(viper.GetString("PRICE")) }
