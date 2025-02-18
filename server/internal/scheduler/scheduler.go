package scheduler

import (
	"fundamental/server/internal/scraping"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Scheduler manages periodic execution of spiders
type Scheduler struct {
	spiderManager *scraping.SpiderManager
	logger        *logrus.Logger
	stopChan      chan struct{}
	wg            sync.WaitGroup
	cities        []string
}

// NewScheduler creates a new scheduler
func NewScheduler(spiderManager *scraping.SpiderManager, logger *logrus.Logger, cities []string) *Scheduler {
	if logger == nil {
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetOutput(os.Stdout)
		logger.SetLevel(logrus.InfoLevel)
	}

	return &Scheduler{
		spiderManager: spiderManager,
		logger:        logger,
		stopChan:      make(chan struct{}),
		cities:        cities,
	}
}

// Start begins the scheduled tasks
func (s *Scheduler) Start() {
	s.wg.Add(2) // One for active spider, one for sold spider

	// Start active spider scheduler (every hour)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.runActiveSpiders()
			}
		}
	}()

	// Start sold spider scheduler (every day at 00:00)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.stopChan:
				return
			default:
				now := time.Now()
				nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
				timer := time.NewTimer(nextRun.Sub(now))

				select {
				case <-s.stopChan:
					timer.Stop()
					return
				case <-timer.C:
					s.runSoldSpiders()
				}
			}
		}
	}()

	// Run immediately on start
	s.runActiveSpiders()
	s.runSoldSpiders()
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// runActiveSpiders runs the active spider for all configured cities
func (s *Scheduler) runActiveSpiders() {
	s.logger.Info("Starting scheduled active spider run")
	for _, city := range s.cities {
		if err := s.spiderManager.RunActiveSpider(city, nil); err != nil {
			s.logger.WithError(err).WithField("city", city).Error("Failed to run active spider")
		}
	}
}

// runSoldSpiders runs the sold spider for all configured cities
func (s *Scheduler) runSoldSpiders() {
	s.logger.Info("Starting scheduled sold spider run")
	for _, city := range s.cities {
		if err := s.spiderManager.RunSoldSpider(city, nil, true); err != nil {
			s.logger.WithError(err).WithField("city", city).Error("Failed to run sold spider")
		}
	}
}
