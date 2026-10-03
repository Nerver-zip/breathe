package timezone

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // Embed IANA timezone database so time.LoadLocation always works offline
)

var (
	mu           sync.Mutex
	detectedName string
	detectedLoc  *time.Location
)

func init() {
	// Auto-detect and configure time.Local when package is loaded.
	_, _ = DetectAndSet("")
}

// Detect determines the local timezone from the environment, system properties, or system clock.
func Detect() (*time.Location, string, error) {
	// 1. Check TZ environment variable
	if tz := strings.TrimSpace(os.Getenv("TZ")); tz != "" {
		tz = strings.TrimPrefix(tz, ":")
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc, tz, nil
		}
	}

	// 2. Check Android system property (persist.sys.timezone)
	if tz := getAndroidTimezone(); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc, tz, nil
		}
	}

	// 3. Check Termux / Unix symlinks (/etc/localtime, $PREFIX/etc/localtime)
	candidates := []string{
		filepath.Join(os.Getenv("PREFIX"), "etc", "localtime"),
		"/data/data/com.termux/files/usr/etc/localtime",
		"/etc/localtime",
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if target, err := os.Readlink(p); err == nil && target != "" {
			if idx := strings.Index(target, "zoneinfo/"); idx != -1 {
				tzName := target[idx+len("zoneinfo/"):]
				if loc, err := time.LoadLocation(tzName); err == nil {
					return loc, tzName, nil
				}
			}
		}
	}

	// 4. Check /etc/timezone (Debian/Ubuntu)
	if content, err := os.ReadFile("/etc/timezone"); err == nil {
		tzName := strings.TrimSpace(string(content))
		if tzName != "" {
			if loc, err := time.LoadLocation(tzName); err == nil {
				return loc, tzName, nil
			}
		}
	}

	// 5. If Go stdlib time.Local has a non-UTC name or non-zero offset, keep it
	if time.Local != nil {
		name := time.Local.String()
		_, offset := time.Now().In(time.Local).Zone()
		if name != "UTC" && name != "" && (name != "Local" || offset != 0) {
			return time.Local, name, nil
		}
	}

	// 6. Fallback: query system date offset via `date +%z`
	if loc, name, err := getTimezoneFromDateCommand(); err == nil {
		return loc, name, nil
	}

	// Default fallback to time.Local (or UTC)
	return time.Local, "Local", nil
}

// getAndroidTimezone checks for Android persist.sys.timezone using getprop
func getAndroidTimezone() string {
	paths := []string{"getprop", "/system/bin/getprop"}
	for _, p := range paths {
		out, err := exec.Command(p, "persist.sys.timezone").Output()
		if err == nil {
			val := strings.TrimSpace(string(out))
			if val != "" {
				return val
			}
		}
	}
	return ""
}

// getTimezoneFromDateCommand runs `date +%z` to construct a FixedZone
func getTimezoneFromDateCommand() (*time.Location, string, error) {
	out, err := exec.Command("date", "+%z").Output()
	if err != nil {
		return nil, "", err
	}
	s := strings.TrimSpace(string(out))
	if len(s) == 5 && (s[0] == '+' || s[0] == '-') {
		sign := 1
		if s[0] == '-' {
			sign = -1
		}
		h, err1 := strconv.Atoi(s[1:3])
		m, err2 := strconv.Atoi(s[3:5])
		if err1 == nil && err2 == nil {
			offsetSeconds := sign * (h*3600 + m*60)
			loc := time.FixedZone(s, offsetSeconds)
			return loc, s, nil
		}
	}
	return nil, "", fmt.Errorf("unrecognized date offset: %s", s)
}

// Set sets the local timezone, either from a specific name or auto-detecting if empty or "auto".
func Set(name string) (*time.Location, error) {
	mu.Lock()
	defer mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "auto") {
		loc, detName, err := Detect()
		if err != nil {
			return nil, err
		}
		time.Local = loc
		detectedName = detName
		detectedLoc = loc
		return loc, nil
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", name, err)
	}
	time.Local = loc
	detectedName = name
	detectedLoc = loc
	return loc, nil
}

// DetectAndSet runs detection and sets time.Local. If explicit is provided, uses it.
func DetectAndSet(explicit string) (*time.Location, error) {
	return Set(explicit)
}

// Current returns the currently active local Location and its identifier string.
func Current() (*time.Location, string) {
	mu.Lock()
	defer mu.Unlock()
	if detectedLoc != nil {
		return detectedLoc, detectedName
	}
	return time.Local, time.Local.String()
}
