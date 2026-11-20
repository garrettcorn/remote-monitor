package config

import (
	"os"
	"testing"
)

func withHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

func TestLoadSaveRoundTrip(t *testing.T) {
	withHome(t)
	c, err := Load() // no file -> defaults
	if err != nil || c.Port != 9120 || c.SettingsPort != 9121 || !c.UpdateCheck {
		t.Fatalf("defaults: %+v %v", c, err)
	}
	if err := c.Set("port", "9200"); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("read_only", "true"); err != nil {
		t.Fatal(err)
	}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil || got.Port != 9200 || !got.ReadOnly {
		t.Fatalf("reloaded: %+v %v", got, err)
	}
	if v, _ := got.Get("port"); v != "9200" {
		t.Fatalf("Get(port) = %q", v)
	}
}

func TestSetValidation(t *testing.T) {
	c := Defaults()
	for _, kv := range [][2]string{{"port", "abc"}, {"port", "70000"}, {"read_only", "maybe"}, {"settings_port", "9120"}, {"name", " "}, {"web_url", "ftp://x"}, {"nope", "1"}} {
		cc := c
		if err := cc.Set(kv[0], kv[1]); err == nil {
			t.Errorf("Set(%s=%s) should fail", kv[0], kv[1])
		}
	}
}

func TestMissingFieldsKeepDefaults(t *testing.T) {
	withHome(t)
	os.MkdirAll(Dir(), 0o700)
	os.WriteFile(Path(), []byte("\xef\xbb\xbf{\"name\":\"Gaming PC\"}"), 0o600)
	c, err := Load()
	if err != nil || c.Name != "Gaming PC" || c.Port != 9120 || c.WebURL != DefaultWebURL {
		t.Fatalf("got %+v %v", c, err)
	}
}
