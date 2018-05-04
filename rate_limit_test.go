package ratelimit

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReadContextKey(t *testing.T) {
	checks := []struct {
		setKey   string
		setValue string
		getKey   string
		expected string
	}{
		{setKey: "SiteKey", setValue: "issuer", getKey: "SiteKey", expected: "issuer"},
		{setKey: "", setValue: "", getKey: "SiteKey", expected: "unknown"},
	}

	for _, c := range checks {
		context := &gin.Context{}
		if len(c.setKey) > 0 {
			context.Set(c.setKey, c.setValue)
		}

		varyByFunc := readContextKey(c.getKey)
		got := varyByFunc(context)
		if got != c.expected {
			t.Errorf("readContextKey test failed (expected: %s, got: %s)", c.expected, got)
		}
	}
}
