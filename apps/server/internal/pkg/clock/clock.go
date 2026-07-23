package clock

import "time"

func NowUTC() time.Time {
	return time.Now().UTC()
}

func NowRFC3339() string {
	return NowUTC().Format(time.RFC3339Nano)
}

func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
