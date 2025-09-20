package utils

import (
	"fmt"
	"time"
)

// PrettyDuration returns string information about duration in pretty format
// Examples:
// 15s                  => "15 seconds"
// 1m20s                => "1 minute 20 seconds"
// 1h15m0s              => "1 hour 15 minutes"
// 1h30m0s              => "1 hour 30 minutes"
// 2m3s                 => "2 minutes 3 seconds"
// 1h1m12s              => "1 hour 1 minute 12 seconds"
// 0s                   => "0 seconds"
func PrettyDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var parts []string

	if hours > 0 {
		unit := "hour"
		if hours > 1 {
			unit = "hours"
		}
		parts = append(parts, fmt.Sprintf("%d %s", hours, unit))
	}

	if minutes > 0 {
		unit := "minute"
		if minutes > 1 {
			unit = "minutes"
		}
		parts = append(parts, fmt.Sprintf("%d %s", minutes, unit))
	}

	if seconds > 0 || len(parts) == 0 {
		unit := "second"
		if seconds != 1 {
			unit = "seconds"
		}
		parts = append(parts, fmt.Sprintf("%d %s", seconds, unit))
	}

	return joinParts(parts)
}

func joinParts(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " " + parts[1]
	default:
		return parts[0] + " " + parts[1] + " " + parts[2]
	}
}
