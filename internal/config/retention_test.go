package config

import (
	"testing"
	"time"
)

func TestRetainUntil(t *testing.T) {
	now := time.Date(2025, 2, 26, 12, 0, 0, 0, time.UTC)

	t.Run("nil retention returns zero", func(t *testing.T) {
		got := RetainUntil(now, nil)
		if !got.IsZero() {
			t.Errorf("RetainUntil(nil) = %v, want zero", got)
		}
	})

	t.Run("zero retention returns zero", func(t *testing.T) {
		r := &RetentionConfig{Days: 0, Weeks: 0, Months: 0}
		got := RetainUntil(now, r)
		if !got.IsZero() {
			t.Errorf("RetainUntil(zero) = %v, want zero", got)
		}
	})

	t.Run("days only", func(t *testing.T) {
		r := &RetentionConfig{Days: 30, Weeks: 0, Months: 0}
		got := RetainUntil(now, r)
		want := time.Date(2025, 1, 27, 12, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Errorf("RetainUntil(days=30) = %v, want %v", got, want)
		}
	})

	t.Run("weeks override when larger", func(t *testing.T) {
		r := &RetentionConfig{Days: 7, Weeks: 2, Months: 0} // 14 days > 7
		got := RetainUntil(now, r)
		want := now.AddDate(0, 0, -14)
		if !got.Equal(want) {
			t.Errorf("RetainUntil(weeks=2) = %v, want %v", got, want)
		}
	})

	t.Run("months override when largest", func(t *testing.T) {
		r := &RetentionConfig{Days: 30, Weeks: 4, Months: 2} // 60 days from months
		got := RetainUntil(now, r)
		want := now.AddDate(0, 0, -60)
		if !got.Equal(want) {
			t.Errorf("RetainUntil(months=2) = %v, want %v", got, want)
		}
	})
}

func TestIsExpired(t *testing.T) {
	now := time.Date(2025, 2, 26, 12, 0, 0, 0, time.UTC)
	r := &RetentionConfig{Days: 30, Weeks: 0, Months: 0}
	cutoff := now.AddDate(0, 0, -30)

	t.Run("recent backup not expired", func(t *testing.T) {
		recent := now.Add(-24 * time.Hour)
		if IsExpired(recent, now, r) {
			t.Error("recent backup should not be expired")
		}
	})

	t.Run("old backup expired", func(t *testing.T) {
		old := cutoff.Add(-time.Hour)
		if !IsExpired(old, now, r) {
			t.Error("old backup should be expired")
		}
	})

	t.Run("nil retention never expired", func(t *testing.T) {
		old := now.AddDate(-1, 0, 0)
		if IsExpired(old, now, nil) {
			t.Error("with nil retention, backup should not be expired")
		}
	})

	t.Run("backup exactly at cutoff not expired", func(t *testing.T) {
		atCutoff := cutoff
		if IsExpired(atCutoff, now, r) {
			t.Error("backup at cutoff time should not be expired (retained)")
		}
	})

	t.Run("backup just before cutoff expired", func(t *testing.T) {
		justBefore := cutoff.Add(-time.Second)
		if !IsExpired(justBefore, now, r) {
			t.Error("backup just before cutoff should be expired")
		}
	})
}
