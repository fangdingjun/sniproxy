package main

import (
	"fmt"
	"strings"
)

type forwardRules map[string]string

func (fr forwardRules) Get(uri string) string {
	if fw, ok := fr[uri]; ok {
		return fw
	}
	return ""
}

func (fr forwardRules) GetN(uri string, port int) string {
	fw := fr.Get(uri)
	if fw != "" {
		return fw
	}

	if fw = fr.Get(fmt.Sprintf("*:%d", port)); fw != "" {
		fw = strings.Replace(fw, "*", uri, 1)
		return fw
	}
	return ""
}

type conf struct {
	Timeout      int
	Listen       []int
	Default      string
	ForwardRules forwardRules `yaml:"forward_rules"`
}
