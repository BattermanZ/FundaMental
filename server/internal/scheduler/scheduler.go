package scheduler

import (
	"bufio"
	"fmt"
	"fundamental/server/config"
	"fundamental/server/internal/scraping"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// JobType represents different types of spider jobs
type JobType int

const (
	JobTypeActive JobType = iota
	JobTypeSold
	JobTypeRefresh
)

// String returns the string representation of a JobType
func (j JobType) String() string {
	switch j {
	case JobTypeActive:
		return "active"
	case JobTypeSold:
		return "sold"
	case JobTypeRefresh:
		return "refresh"
	default:
		return "unknown"
	}
}

// Scheduler manages periodic execution of spiders
type Scheduler struct {
	spiderManager *scraping.SpiderManager
	logger        *logrus.Logger
	stopChan      chan struct{}
	wg            sync.WaitGroup
	cities        []string
	jobMutex      sync.Mutex // Ensures sequential job execution
	isStartupRun  bool       // Tracks whether we're in startup run
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
		isStartupRun:  true, // Initialize as true for startup
	}
}

// Start begins the scheduled tasks
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.runScheduler()
}

// runScheduler handles all scheduled tasks
func (s *Scheduler) runScheduler() {
	defer s.wg.Done()

	// Run startup jobs in a separate goroutine
	go func() {
		s.jobMutex.Lock()
		defer s.jobMutex.Unlock()
		s.logger.Info("Running startup spider jobs")
		s.runActiveSpiders()
		s.isStartupRun = false // Mark startup as complete
		s.logger.Info("Startup spider jobs completed")
	}()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case t := <-ticker.C:
			s.executeScheduledJobs(t)
		}
	}
}

// executeScheduledJobs runs all jobs that are scheduled for the given time
func (s *Scheduler) executeScheduledJobs(t time.Time) {
	// Skip if we're still running startup jobs
	if s.isStartupRun {
		s.logger.Debug("Skipping scheduled jobs while startup is in progress")
		return
	}

	s.jobMutex.Lock()
	defer s.jobMutex.Unlock()

	s.logger.WithFields(logrus.Fields{
		"hour":   t.Hour(),
		"minute": t.Minute(),
	}).Debug("Checking scheduled jobs")

	// Check if it's time for the sold spider (midnight)
	if t.Hour() == 0 && t.Minute() == 0 {
		s.logger.Info("Starting scheduled sold spider jobs")
		s.runSoldSpiders()
		s.logger.Info("Completed scheduled sold spider jobs")
	}

	// Check if it's time for the active spider (every hour)
	if t.Minute() == 0 {
		s.logger.Info("Starting scheduled active spider jobs")
		s.runActiveSpiders()
		s.logger.Info("Completed scheduled active spider jobs")
	}

	// Check refresh schedule
	s.checkAndRunRefreshSpiders(t)
}

// runActiveSpiders runs the active spider for all configured cities sequentially
func (s *Scheduler) runActiveSpiders() {
	s.logger.Info("Starting active spider run")
	for _, city := range s.cities {
		normalizedCity := config.NormalizeCity(city)
		s.logger.WithFields(logrus.Fields{
			"city":            city,
			"normalized_city": normalizedCity,
			"job_type":        JobTypeActive.String(),
		}).Info("Starting spider job")

		if err := s.spiderManager.RunActiveSpider(normalizedCity, nil); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"city":            city,
				"normalized_city": normalizedCity,
				"job_type":        JobTypeActive.String(),
			}).Error("Spider job failed")
		} else {
			s.logger.WithFields(logrus.Fields{
				"city":            city,
				"normalized_city": normalizedCity,
				"job_type":        JobTypeActive.String(),
			}).Info("Spider job completed successfully")
		}
	}
}

// runSoldSpiders runs the sold spider for all configured cities sequentially
func (s *Scheduler) runSoldSpiders() {
	s.logger.Info("Starting sold spider run")
	for _, city := range s.cities {
		normalizedCity := config.NormalizeCity(city)
		s.logger.WithFields(logrus.Fields{
			"city":            city,
			"normalized_city": normalizedCity,
			"job_type":        JobTypeSold.String(),
		}).Info("Starting spider job")

		if err := s.spiderManager.RunSoldSpider(normalizedCity, nil, true); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"city":            city,
				"normalized_city": normalizedCity,
				"job_type":        JobTypeSold.String(),
			}).Error("Spider job failed")
		} else {
			s.logger.WithFields(logrus.Fields{
				"city":            city,
				"normalized_city": normalizedCity,
				"job_type":        JobTypeSold.String(),
			}).Info("Spider job completed successfully")
		}
	}
}

// checkAndRunRefreshSpiders checks and runs refresh spiders for the current time
func (s *Scheduler) checkAndRunRefreshSpiders(t time.Time) {
	if t.Minute() != 0 { // Only check on the hour
		return
	}

	timeSlots := []int{0, 4, 8, 12, 16, 20}
	daysOfWeek := []time.Weekday{
		time.Sunday,
		time.Monday,
		time.Tuesday,
		time.Wednesday,
		time.Thursday,
		time.Friday,
		time.Saturday,
	}

	// Create schedule slots
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

	// Check each city's schedule
	for city, slot := range citySchedule {
		if t.Weekday() == slot.day && t.Hour() == slot.hour {
			normalizedCity := config.NormalizeCity(city)
			s.logger.WithFields(logrus.Fields{
				"city":            city,
				"normalized_city": normalizedCity,
				"job_type":        JobTypeRefresh.String(),
				"day":             slot.day,
				"hour":            slot.hour,
			}).Info("Starting spider job")

			if err := s.spiderManager.RunRefreshSpider(normalizedCity); err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"city":            city,
					"normalized_city": normalizedCity,
					"job_type":        JobTypeRefresh.String(),
				}).Error("Spider job failed")
			} else {
				s.logger.WithFields(logrus.Fields{
					"city":            city,
					"normalized_city": normalizedCity,
					"job_type":        JobTypeRefresh.String(),
				}).Info("Spider job completed successfully")
			}
		}
	}
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

func (s *Scheduler) startSpiderForCity(city string) error {
	normalizedCity := config.NormalizeCity(city)
	s.logger.Infof("Starting spider for city: %s (normalized: %s)", city, normalizedCity)

	// Create spider command with normalized city name
	cmd := exec.Command("python3", "server/scripts/run_spider.py")
	cmd.Stdin = strings.NewReader(fmt.Sprintf(`{"spider_type": "active", "place": "%s", "original_city": "%s"}`, normalizedCity, city))

	// Set up pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start spider: %v", err)
	}

	// Handle output in goroutines
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			s.logger.Info(scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			s.logger.Error(scanner.Text())
		}
	}()

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("spider failed: %v", err)
	}

	return nil
}
