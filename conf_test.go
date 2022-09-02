package main

import (
	"os"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestConf(t *testing.T) {
	data, err := os.ReadFile("config.sample.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var c conf
	if err := yaml.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	// fmt.Printf("%+v\n", c)
	var testdata = map[string]string{
		"www.example.com": "127.0.0.1:8443 proxy-v2",
		"b.example.com":   "127.0.0.1:8542",
	}
	r := forwardRules(c.ForwardRules)
	for k, v := range testdata {
		s := r.Get(k)
		if s != v {
			t.Errorf("expected: %s, got: %s", v, s)
		}
	}

	if r.GetN("a.com", 9999) != "a.com:443" {
		t.Errorf("expected a.com:9999, got %s", r.GetN("a.com", 9999))
	}
}
