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
	s.wg.Add(3) // One for active spider, one for sold spider, one for refresh spider

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

	// Start refresh spider scheduler
	go func() {
		defer s.wg.Done()
		s.scheduleRefreshSpiders()
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

// scheduleRefreshSpiders schedules refresh operations for each city
func (s *Scheduler) scheduleRefreshSpiders() {
	timeSlots := []int{0, 4, 8, 12, 16, 20} // Hours of the day for scheduling
	daysOfWeek := []time.Weekday{
		time.Sunday,
		time.Monday,
		time.Tuesday,
		time.Wednesday,
		time.Thursday,
		time.Friday,
		time.Saturday,
	}

	// Create a schedule that prioritizes earlier time slots
	type scheduleSlot struct {
		day  time.Weekday
		hour int
	}

	var schedule []scheduleSlot
	// First fill all midnight slots
	for _, day := range daysOfWeek {
		schedule = append(schedule, scheduleSlot{day, timeSlots[0]})
	}
	// Then fill all 4am slots
	for _, day := range daysOfWeek {
		schedule = append(schedule, scheduleSlot{day, timeSlots[1]})
	}
	// Continue for each time slot
	for _, hour := range timeSlots[2:] {
		for _, day := range daysOfWeek {
			schedule = append(schedule, scheduleSlot{day, hour})
		}
	}

	// Assign cities to schedule slots
	citySchedule := make(map[string]scheduleSlot)
	for i, city := range s.cities {
		if i < len(schedule) {
			citySchedule[city] = schedule[i]
		}
	}

	// Run the scheduler
	for {
		select {
		case <-s.stopChan:
			return
		default:
			now := time.Now()

			// Check each city's schedule
			for city, slot := range citySchedule {
				if now.Weekday() == slot.day && now.Hour() == slot.hour && now.Minute() == 0 {
					s.logger.WithFields(logrus.Fields{
						"city": city,
						"day":  slot.day,
						"hour": slot.hour,
					}).Info("Running scheduled refresh spider")

					if err := s.spiderManager.RunRefreshSpider(city); err != nil {
						s.logger.WithError(err).WithField("city", city).Error("Failed to run refresh spider")
					}
				}
			}

			// Sleep for a minute before checking again
			time.Sleep(time.Minute)
		}
	}
}
