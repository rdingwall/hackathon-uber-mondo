package main

import (
	"testing"
	"log"
)

func TestRandomCarEmoji(t *testing.T) {

	car1 := randomCarEmoji()
	car2 := randomCarEmoji()
	car3 := randomCarEmoji()

	log.Printf(car1)
	log.Printf(car2)
	log.Printf(car3)
}