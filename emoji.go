package main
import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var carEmojis = []string {"🚘", "🚖", "🚗" }

func randomCarEmoji() string {
	return carEmojis[rand.Intn(len(carEmojis))]
}