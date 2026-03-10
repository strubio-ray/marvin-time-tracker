package main

import (
	"context"
	"log"
	"time"
)

const renewalThreshold = 7*time.Hour + 45*time.Minute

type Renewal struct {
	store    *StateStore
	notifier Notifier
	broker   BrokerPublisher
	stop     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	now      func() time.Time // for testing
}

func NewRenewal(store *StateStore, notifier Notifier, broker BrokerPublisher) *Renewal {
	ctx, cancel := context.WithCancel(context.Background())
	return &Renewal{
		store:    store,
		notifier: notifier,
		broker:   broker,
		stop:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
		now:      time.Now,
	}
}

func (rn *Renewal) Start() {
	go rn.run()
}

func (rn *Renewal) Stop() {
	rn.cancel()
	close(rn.stop)
}

func (rn *Renewal) run() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rn.stop:
			return
		case <-ticker.C:
			rn.check()
		}
	}
}

func (rn *Renewal) check() {
	state := rn.store.Get()
	if !state.IsTracking() {
		return
	}

	if state.LiveActivityStartedAt.IsZero() {
		return
	}

	elapsed := rn.now().Sub(state.LiveActivityStartedAt)
	if elapsed < renewalThreshold {
		return
	}

	if rn.notifier == nil {
		return
	}

	log.Printf("renewal: Live Activity at %v, triggering renewal", elapsed.Round(time.Second))

	// End current Live Activity
	if state.UpdateToken != "" {
		if err := rn.notifier.EndActivity(state.UpdateToken); err != nil {
			log.Printf("renewal: end error: %v", err)
			return
		}
	}

	// Brief pause for APNs processing
	time.Sleep(500 * time.Millisecond)

	// Start new Live Activity with original startedAt
	rn.store.Update(func(s *State) {
		s.LiveActivityStartedAt = rn.now()
		s.UpdateToken = "" // Will be re-registered by iOS app
	})

	tokens, err := rn.store.ConsumeNotifyTokens()
	if err != nil {
		log.Printf("renewal: failed to consume tokens: %v", err)
	}
	notifyTrackingStarted(rn.ctx, tokens, rn.notifier, rn.broker, state.TaskTitle, state.StartedAt, DefaultSilentPushGracePeriod, func() string {
		s := rn.store.Get()
		if !s.IsTracking() {
			return "stopped"
		}
		return s.UpdateToken
	})
}
