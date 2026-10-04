package main

import "testing"

func TestCustomEndpointValidationBeforePersisting(t *testing.T) {
	good := customProvider{ID: "1234567890", Name: "Test", CountryCC: "US", Base: "https://example.com/speed/", Download: "backend/garbage.php", Upload: "backend/empty.php", Ping: "backend/empty.php"}
	if err := validateCustomProvider(good); err != nil {
		t.Fatal(err)
	}
	for _, base := range []string{"file:///tmp/test", "https://user:password@example.com/", "https://example.com/#fragment", "http://", "https://example.com/%zz"} {
		p := good
		p.Base = base
		if validateCustomProvider(p) == nil {
			t.Errorf("accepted invalid base %q", base)
		}
	}
	for _, path := range []string{"https://other.example.com/", "//other.example.com/", "empty.php\n", ""} {
		p := good
		p.Ping = path
		if validateCustomProvider(p) == nil {
			t.Errorf("accepted invalid endpoint %q", path)
		}
	}
	p := good
	p.CountryCC = "USA"
	if validateCustomProvider(p) == nil {
		t.Error("accepted invalid country code")
	}
}
