package schedule

import (
	"fmt"
	"time"

	"VelBackuper/internal/config"
)

// NextRun returns the next run time after 'now' for the given schedule, and a short description.
// Uses the same logic as systemd OnCalendar (day: 02:00, 14:00, etc.; week: Mon/Fri 02:00; month: day 1/15 02:00).
func NextRun(s *config.ScheduleConfig, now time.Time) (next time.Time, desc string) {
	if s == nil || s.Times < 1 {
		return time.Time{}, "no schedule"
	}
	times := s.Times
	if times > 5 {
		times = 5
	}

	// All runs are at 02:00 local in systemd; we use UTC for simplicity
	hour, minute := 2, 0
	jitterMin := s.JitterMinutes
	if jitterMin < 0 {
		jitterMin = 0
	}

	switch s.Period {
	case "week":
		// weekdays: Mon=1..Fri=5, 02:00
		weekdays := [][]int{{1}, {1, 4}, {1, 3, 5}, {1, 2, 4, 5}, {1, 2, 3, 4, 5}}[times-1]
		wd := int(now.Weekday()) // Sun=0, Mon=1, ...
		if wd == 0 {
			wd = 7
		}
		for _, d := range weekdays {
			daysAhead := d - wd
			if daysAhead <= 0 {
				daysAhead += 7
			}
			cand := now.AddDate(0, 0, daysAhead)
			cand = time.Date(cand.Year(), cand.Month(), cand.Day(), hour, minute, 0, 0, now.Location())
			if cand.After(now) {
				return cand.Add(time.Duration(jitterMin) * time.Minute), fmt.Sprintf("weekly %d×", times)
			}
		}
		cand := now.AddDate(0, 0, 7)
		cand = time.Date(cand.Year(), cand.Month(), cand.Day(), hour, minute, 0, 0, now.Location())
		return cand.Add(time.Duration(jitterMin) * time.Minute), fmt.Sprintf("weekly %d×", times)

	case "month":
		days := [][]int{{1}, {1, 15}, {1, 10, 20}, {1, 8, 15, 22}, {1, 7, 14, 21, 28}}[times-1]
		y, m, _ := now.Date()
		for _, day := range days {
			var cand time.Time
			if day <= 28 {
				cand = time.Date(y, m, day, hour, minute, 0, 0, now.Location())
			} else {
				// last day of month
				cand = time.Date(y, m+1, 0, hour, minute, 0, 0, now.Location())
			}
			if cand.After(now) {
				return cand.Add(time.Duration(jitterMin) * time.Minute), fmt.Sprintf("monthly %d×", times)
			}
		}
		cand := time.Date(y, m+1, days[0], hour, minute, 0, 0, now.Location())
		return cand.Add(time.Duration(jitterMin) * time.Minute), fmt.Sprintf("monthly %d×", times)

	default:
		// day: 02:00, 14:00, etc.
		hours := [][]int{{2}, {2, 14}, {2, 10, 18}, {2, 8, 14, 20}, {2, 6, 12, 18, 22}}[times-1]
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for _, h := range hours {
			cand := today.Add(time.Duration(h)*time.Hour + time.Duration(minute)*time.Minute)
			if cand.After(now) {
				return cand.Add(time.Duration(jitterMin) * time.Minute), fmt.Sprintf("daily %d×", times)
			}
		}
		tomorrow := today.AddDate(0, 0, 1)
		cand := tomorrow.Add(time.Duration(hours[0])*time.Hour + time.Duration(minute)*time.Minute)
		return cand.Add(time.Duration(jitterMin) * time.Minute), fmt.Sprintf("daily %d×", times)
	}
}
