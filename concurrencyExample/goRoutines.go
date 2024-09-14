package concurrencyExample

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	iterCount int = 1000
)

var (
	m       = sync.RWMutex{}
	wg      = sync.WaitGroup{}
	dbData  = []string{"id1", "id2", "id3", "id4", "id5", "id6", "id7", "id8", "id9", "id10"}
	results []string
)

func DoDbCall() {
	t0 := time.Now()
	wg.Add(iterCount)
	for i := 0; i < iterCount; i++ {
		go dbCall(i % len(dbData))
	}
	wg.Wait()

	slog.Info("Done!", "time", time.Since(t0))
}

func DoBigCount() {
	t0 := time.Now()
	wg.Add(iterCount)
	for i := 0; i < iterCount; i++ {
		go count()
	}
	wg.Wait()
	slog.Info("Done!", "time", time.Since(t0))
}

func dbCall(i int) {
	defer wg.Done()
	slog.Info("Calling the database", "target record", i)
	// var delay = rand.Intn(2000)
	var delay = 2000
	time.Sleep(time.Duration(delay) * time.Millisecond)
	fmt.Println("The result from the database is: ", dbData[i])
	save(dbData[i])
	slog.Info("Retrieved from database", "result #", i, "result", dbData[i])
}

func save(result string) {
	m.Lock()
	results = append(results, result)
	m.Unlock()
}

func count() {
	var res int
	for i := 0; i < 100e6; i++ {
		res += 1
	}
	wg.Done()
}
