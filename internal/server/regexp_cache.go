package server

import (
	"regexp"
	"sync"
)

var regexpCache sync.Map

func cachedRegexp(pattern string) *regexp.Regexp {
	if re, ok := regexpCache.Load(pattern); ok {
		return re.(*regexp.Regexp)
	}
	re := regexp.MustCompile(pattern)
	actual, _ := regexpCache.LoadOrStore(pattern, re)
	return actual.(*regexp.Regexp)
}
