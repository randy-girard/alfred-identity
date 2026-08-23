package updatecheck

import (
	"strconv"
	"strings"
)

// IsNewer reports whether candidate is a higher version than current.
// Both may include an optional "v" prefix and an optional pre-release suffix
// (e.g. v1.2.0-beta.1). Build metadata (+...) is ignored. Returns false when
// either side is empty or not a parseable semver-like version.
func IsNewer(candidate, current string) bool {
	return compareVersions(candidate, current) > 0
}

// compareVersions returns -1 if a < b, 0 if equal, +1 if a > b.
func compareVersions(a, b string) int {
	va, oka := parseVersion(a)
	vb, okb := parseVersion(b)
	if !oka || !okb {
		return 0
	}
	for i := 0; i < 3; i++ {
		if va.core[i] < vb.core[i] {
			return -1
		}
		if va.core[i] > vb.core[i] {
			return 1
		}
	}
	// Semver: release (no pre) > any pre-release with the same core.
	if va.pre == "" && vb.pre == "" {
		return 0
	}
	if va.pre == "" {
		return 1
	}
	if vb.pre == "" {
		return -1
	}
	return comparePre(va.pre, vb.pre)
}

type version struct {
	core [3]int
	pre  string
}

func parseVersion(s string) (version, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	if s == "" {
		return version{}, false
	}
	// Drop build metadata.
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	pre := ""
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
		if pre == "" {
			return version{}, false
		}
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return version{}, false
	}
	var v version
	v.pre = pre
	for i := 0; i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 {
			return version{}, false
		}
		v.core[i] = n
	}
	return v, true
}

func comparePre(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) < n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		ai, aNum := strconv.Atoi(as[i])
		bi, bNum := strconv.Atoi(bs[i])
		if aNum == nil && bNum == nil {
			if ai < bi {
				return -1
			}
			if ai > bi {
				return 1
			}
			continue
		}
		// Numeric identifiers have lower precedence than non-numeric.
		if aNum == nil {
			return -1
		}
		if bNum == nil {
			return 1
		}
		if c := strings.Compare(as[i], bs[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(as) < len(bs):
		return -1
	case len(as) > len(bs):
		return 1
	default:
		return 0
	}
}
