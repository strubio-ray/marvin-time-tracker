package main

import (
	"context"
	"log"
	"time"
)

type Poller struct {
	marvin         MarvinAPIClient
	store          *StateStore
	notifier       Notifier
	activeInterval time.Duration
	idleInterval   time.Duration
	quota          *QuotaCounter
	stop           chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewPoller(marvin MarvinAPIClient, store *StateStore, notifier Notifier, activeInterval, idleInterval time.Duration, quota *QuotaCounter) *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Poller{
		marvin:         marvin,
		store:          store,
		notifier:       notifier,
		activeInterval: activeInterval,
		idleInterval:   idleInterval,
		quota:          quota,
		stop:           make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (p *Poller) Start() {
	go p.run()
}

func (p *Poller) Stop() {
	p.cancel()
	close(p.stop)
}

func (p *Poller) run() {
	ticker := time.NewTicker(p.idleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			state := p.store.Get()

			// Skip if recent webhook
			interval := p.currentInterval(state.IsTracking())
			if !state.LastWebhookAt.IsZero() && time.Since(state.LastWebhookAt) < interval {
				log.Printf("poller: skipping, recent webhook")
				continue
			}

			if p.quota != nil && p.quota.IsExhausted() {
				log.Printf("poller: quota exhausted, skipping")
				continue
			}

			p.poll()

			// Adapt interval
			newState := p.store.Get()
			newInterval := p.currentInterval(newState.IsTracking())
			if newInterval != interval {
				ticker.Reset(newInterval)
				log.Printf("poller: interval changed to %v", newInterval)
			}
		}
	}
}

func (p *Poller) currentInterval(tracking bool) time.Duration {
	if tracking {
		return p.activeInterval
	}
	return p.idleInterval
}

func (p *Poller) poll() {
	if p.quota != nil {
		p.quota.Increment()
	}

	item, err := p.marvin.GetTrackedItem()
	if err != nil {
		log.Printf("poller: error: %v", err)
		return
	}

	p.store.Update(func(s *State) {
		s.LastPollAt = time.Now()
	})

	state := p.store.Get()

	if item != nil && !state.IsTracking() && (state.LastStopAt.IsZero() || item.StartedAt >= state.LastStopAt.UnixMilli()) {
		// Missed start
		log.Printf("poller: detected missed start for %s", item.TaskID)
		p.store.Update(func(s *State) {
			s.TrackingTaskID = item.TaskID
			s.TaskTitle = item.Title
			s.StartedAt = item.StartedAt
			s.LiveActivityStartedAt = time.Now()
		})

		notifyTrackingStarted(p.ctx, p.store, p.notifier, item.Title, item.StartedAt, DefaultSilentPushGracePeriod)
	} else if item == nil && state.IsTracking() {
		// Missed stop
		log.Printf("poller: detected missed stop")
		updateToken := state.UpdateToken
		p.store.Update(func(s *State) {
			s.TrackingTaskID = ""
			s.TaskTitle = ""
			s.StartedAt = 0
			s.Times = nil
			s.LiveActivityStartedAt = time.Time{}
			s.UpdateToken = ""
		})

		notifyTrackingStopped(p.store, p.notifier, updateToken)
	}
}
