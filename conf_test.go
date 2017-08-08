package main

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"testing"
)

func TestConf(t *testing.T) {
	data, err := ioutil.ReadFile("config.sample.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var c conf
	if err := yaml.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", c)
	var testdata = map[string]string{
		"www.example.com": "127.0.0.1:8443",
		"b.example.com":   "127.0.0.1:8541",
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
