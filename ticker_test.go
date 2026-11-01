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

// lateFiredTimer hands the ticker goroutine a channel that has already fired,
// but only after a pause that lets the test call Stop and begin its receive.
// The goroutine thus reaches its next select with the tick, the stop signal
// and the receiver all ready.
type lateFiredTimer struct{}

func (lateFiredTimer) Start(time.Duration) {}
func (lateFiredTimer) Stop()               {}

func (lateFiredTimer) C() <-chan time.Time {
	time.Sleep(time.Millisecond)
	c := make(chan time.Time, 1)
	c <- time.Now()
	return c
}

func TestTickerNoTickAfterStop(t *testing.T) {
	// A receive started after Stop returns must not get a tick (#198). Without
	// the fix select picks between the tick and stop at random, so repeat to
	// make a miss practically impossible.
	for i := 0; i < 100; i++ {
		ticker := NewTicker(NewConstantBackOff(time.Hour))
		ticker.timer = lateFiredTimer{}
		<-ticker.C // the first tick is guaranteed
		ticker.Stop()
		if _, ok := <-ticker.C; ok {
			t.Fatalf("iteration %d: received a tick after Stop returned", i)
		}
	}
}
