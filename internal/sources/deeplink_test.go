package sources

import "testing"

func TestEncodeParseSourceDeepLinkRoundTrip(t *testing.T) {
	src := Source{Name: "GoodGuys", Host: "identity.goodguysguild.com", Token: "tok_abc-123"}
	raw, err := EncodeSourceDeepLink(src)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseSourceDeepLink(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != src.Name || got.Host != src.Host || got.Token != src.Token {
		t.Fatalf("got %+v", got)
	}
}

func TestParseSourceDeepLinkOpenAlfredAndQuoted(t *testing.T) {
	src := Source{Name: "Guild", Host: "127.0.0.1:8181", Token: "secret"}
	appURL, err := EncodeSourceDeepLink(src)
	if err != nil {
		t.Fatal(err)
	}
	u, err := ParseSourceDeepLink(appURL)
	if err != nil {
		t.Fatal(err)
	}
	httpsURL := "https://identity.example.com/open-alfred?d=" + appURL[len(AppURLScheme+"://"+deepLinkHost+"?d="):]
	got, err := ParseSourceDeepLink(`"` + httpsURL + `"`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != u.Name || got.Host != u.Host || got.Token != u.Token {
		t.Fatalf("https got %+v", got)
	}
	hashURL := "https://identity.example.com/open-alfred#d=" + appURL[len(AppURLScheme+"://"+deepLinkHost+"?d="):]
	got, err = ParseSourceDeepLink(hashURL)
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != src.Token {
		t.Fatalf("hash got %+v", got)
	}
}

func TestParseSourceDeepLinkQueryFields(t *testing.T) {
	got, err := ParseSourceDeepLink("alfred-identity://import?name=Guild&host=example.com:443&token=abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Guild" || got.Host != "example.com:443" || got.Token != "abc" {
		t.Fatalf("%+v", got)
	}
}

func TestParseSourceDeepLinkRejects(t *testing.T) {
	for _, raw := range []string{"", "https://example.com/", "alfred-identity://import", "alfred-identity://import?d=%%%"} {
		if _, err := ParseSourceDeepLink(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestLooksLikeAndDeepLinkFromArgs(t *testing.T) {
	if LooksLikeSourceDeepLink("https://example.com/other") {
		t.Fatal("false positive")
	}
	if !LooksLikeSourceDeepLink("alfred-identity://import?d=x") {
		t.Fatal("scheme")
	}
	if !LooksLikeSourceDeepLink("https://identity.example.com/open-alfred?d=x") {
		t.Fatal("https")
	}
	args := []string{"-psn_0_123", "alfred-identity://import?d=abc"}
	if got := DeepLinkFromArgs(args); got != "alfred-identity://import?d=abc" {
		t.Fatalf("got %q", got)
	}
	if DeepLinkFromArgs(nil) != "" {
		t.Fatal("empty args")
	}
}

func TestEncodeSourceDeepLinkRequiresFields(t *testing.T) {
	if _, err := EncodeSourceDeepLink(Source{Name: "n", Host: "h"}); err == nil {
		t.Fatal("expected token required")
	}
}
