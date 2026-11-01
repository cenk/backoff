package backoff

import (
	"errors"
	"log"
	"testing"
	"time"
)

func TestTicker(t *testing.T) {
	const successOn = 3
	var i = 0

	// This function is successful on "successOn" calls.
	f := func() error {
		i++
		log.Printf("function is called %d. time\n", i)

		if i == successOn {
			log.Println("OK")
			return nil
		}

		log.Println("error")
		return errors.New("error")
	}

	b := NewExponentialBackOff()
	ticker := NewTicker(b)
	ticker.timer = &testTimer{}

	var err error
	for range ticker.C {
		if err = f(); err != nil {
			t.Log(err)
			continue
		}

		break
	}
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if i != successOn {
		t.Errorf("invalid number of retries: %d", i)
	}
}

func TestTickerStop(t *testing.T) {
	// Stop closes the channel and is safe to call more than once.
	ticker := NewTicker(NewConstantBackOff(time.Hour))
	<-ticker.C // the first tick is guaranteed
	ticker.Stop()
	ticker.Stop() // must not panic (sync.Once)
	if _, ok := <-ticker.C; ok {
		t.Error("expected ticker channel to be closed after Stop")
	}
	if ticker.timer.(*defaultTimer).timer.Stop() {
		t.Error("expected ticker timer to be stopped after Stop")
	}
}

func TestTickerStopsOnBackOffStop(t *testing.T) {
	// When the BackOff returns Stop, the ticker closes its channel after the
	// guaranteed first tick.
	ticker := NewTicker(&StopBackOff{})
	defer ticker.Stop()
	if _, ok := <-ticker.C; !ok {
		t.Fatal("expected at least one tick")
	}
	if _, ok := <-ticker.C; ok {
		t.Error("expected channel to close after BackOff returned Stop")
	}
}
