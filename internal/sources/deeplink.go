package sources

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// AppURLScheme is the custom protocol Discord's HTTPS landing page opens.
const AppURLScheme = "alfred-identity"

const (
	deepLinkHost     = "import"
	maxDeepLinkBytes = 4096
)

// EncodeSourceDeepLink builds alfred-identity://import?d=<base64url JSON>.
func EncodeSourceDeepLink(src Source) (string, error) {
	payload, err := encodeSourcePayload(src)
	if err != nil {
		return "", err
	}
	return AppURLScheme + "://" + deepLinkHost + "?d=" + payload, nil
}

func encodeSourcePayload(src Source) (string, error) {
	name := strings.TrimSpace(src.Name)
	host := NormalizeHost(src.Host)
	token := strings.TrimSpace(src.Token)
	if name == "" || host == "" || token == "" {
		return "", fmt.Errorf("name, host, and token required")
	}
	b, err := json.Marshal(map[string]string{
		"name":  name,
		"host":  host,
		"token": token,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// LooksLikeSourceDeepLink reports whether raw is an app or landing-page import URL.
func LooksLikeSourceDeepLink(raw string) bool {
	s := strings.Trim(strings.TrimSpace(raw), `"'`)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, AppURLScheme+":") || strings.Contains(lower, "/open-alfred")
}

// DeepLinkFromArgs returns the first CLI argument that looks like a source import URL.
func DeepLinkFromArgs(args []string) string {
	for _, a := range args {
		if LooksLikeSourceDeepLink(a) {
			return strings.Trim(strings.TrimSpace(a), `"'`)
		}
	}
	return ""
}

// ParseSourceDeepLink reads name/host/token from an alfred-identity:// or /open-alfred URL.
func ParseSourceDeepLink(raw string) (Source, error) {
	raw = strings.Trim(strings.TrimSpace(raw), `"'`)
	if raw == "" {
		return Source{}, fmt.Errorf("empty link")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Source{}, fmt.Errorf("invalid link")
	}
	q := u.Query()
	if d := strings.TrimSpace(q.Get("d")); d != "" {
		return parseSourceDeepLinkPayload(d)
	}
	frag := strings.TrimSpace(u.Fragment)
	if strings.HasPrefix(strings.ToLower(frag), "d=") {
		return parseSourceDeepLinkPayload(strings.TrimSpace(frag[2:]))
	}
	name := strings.TrimSpace(q.Get("name"))
	host := NormalizeHost(q.Get("host"))
	token := strings.TrimSpace(q.Get("token"))
	if name != "" && host != "" && token != "" {
		return Source{Name: name, Host: host, Token: token, Notes: strings.TrimSpace(q.Get("notes"))}, nil
	}
	return Source{}, fmt.Errorf("link is missing source data")
}

func parseSourceDeepLinkPayload(d string) (Source, error) {
	d = strings.TrimSpace(d)
	if d == "" {
		return Source{}, fmt.Errorf("link is missing source data")
	}
	b, err := base64.RawURLEncoding.DecodeString(d)
	if err != nil {
		b, err = base64.URLEncoding.DecodeString(d)
	}
	if err != nil {
		return Source{}, fmt.Errorf("invalid source payload")
	}
	if len(b) == 0 || len(b) > maxDeepLinkBytes {
		return Source{}, fmt.Errorf("invalid source payload")
	}
	parsed, err := ParseImportSources(b)
	if err != nil {
		return Source{}, err
	}
	if len(parsed) == 0 {
		return Source{}, fmt.Errorf("no source found")
	}
	src := parsed[0]
	if strings.TrimSpace(src.Token) == "" {
		return Source{}, fmt.Errorf("token required")
	}
	return src, nil
}
