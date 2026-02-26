package config

import "time"

func RetainUntil(now time.Time, r *RetentionConfig) time.Time {
	if r == nil {
		return time.Time{}
	}
	days := r.Days
	if r.Weeks*7 > days {
		days = r.Weeks * 7
	}
	if r.Months*30 > days {
		days = r.Months * 30
	}
	if days <= 0 {
		return time.Time{}
	}
	return now.AddDate(0, 0, -days)
}

func IsExpired(backupTime, now time.Time, r *RetentionConfig) bool {
	cutoff := RetainUntil(now, r)
	if cutoff.IsZero() {
		return false
	}
	return backupTime.Before(cutoff)
}
