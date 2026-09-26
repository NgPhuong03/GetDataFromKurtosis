package config

import "testing"

func TestNormalizeModel(t *testing.T) {
	cases := map[string]string{
		"baseline": "baseline",
		"DF":       "baseline",
		"df":       "baseline",
		"SRSW":     "srsw",
		"lsw":      "lsw",
		" desw ":   "desw",
	}
	for raw, want := range cases {
		got, err := NormalizeModel(raw)
		if err != nil {
			t.Fatalf("NormalizeModel(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("NormalizeModel(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := NormalizeModel("weighted"); err == nil {
		t.Fatal("expected unknown model to fail")
	}
}

func TestNormalizeDefaults(t *testing.T) {
	cfg := Config{
		Mongo: MongoConfig{MongoURI: "mongodb://localhost", DBName: "kurtosis"},
		Dora:  DoraConfig{URL: "http://127.0.0.1:32818/"},
		Model: "df",
	}
	if err := cfg.normalize(); err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "baseline" {
		t.Fatalf("model = %q", cfg.Model)
	}
	if cfg.Dora.URL != "http://127.0.0.1:32818" {
		t.Fatalf("url = %q", cfg.Dora.URL)
	}
	if cfg.SlotsPerEpoch != 32 || cfg.TargetEpochs != 200 || cfg.TimeLoop != 10 {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

func TestExportConfigAllowsEmptyModel(t *testing.T) {
	cfg := Config{
		Mongo: MongoConfig{MongoURI: "mongodb://localhost", DBName: "kurtosis_desw"},
	}
	if err := cfg.normalize(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.RequireFetch(); err == nil {
		t.Fatal("fetch should require a model and a Dora URL")
	}
}
