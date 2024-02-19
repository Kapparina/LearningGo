package concurrencyExample

import (
	"fmt"
	"math/rand"
	"time"
)

const MaxChickenPrice = 5
const MaxTofuPrice = 3

func FindDeal() {
	var chickenChannel = make(chan string)
	var tofuChannel = make(chan string)
	var websites = []string{"coles.com", "woolworths.com", "aldi.com"}
	for i := range websites {
		go checkChickenPrices(websites[i], chickenChannel)
		go checkTofuPrices(websites[i], tofuChannel)
	}
	sendMessage(chickenChannel, tofuChannel)
}

func checkChickenPrices(website string, chickenChannel chan string) {
	for {
		time.Sleep(time.Second * 1)
		var chickenPrice = rand.Float32() * 20
		if chickenPrice <= MaxChickenPrice {
			chickenChannel <- website
			break
		}
	}
}

func checkTofuPrices(website string, tofuChannel chan string) {
	for {
		time.Sleep(time.Second * 1)
		var tofuPrice = rand.Float32() * 20
		if tofuPrice <= MaxTofuPrice {
			tofuChannel <- website
			break
		}
	}
}
func sendMessage(chickenChannel chan string, tofuChannel chan string) {
	select {
	case website := <-chickenChannel:
		fmt.Printf("\nText sent: Found a deal on chicken at %v", website)
	case website := <-tofuChannel:
		fmt.Printf("\nEmail sent: Found a deal on tofu at %v", website)
	}
}
