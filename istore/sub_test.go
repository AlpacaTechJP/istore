package istore

import (
	"testing"
)

func TestExtractTargetURL(t *testing.T) {
	url1 := extractTargetURL("/abc/http://example.com/foo/bar.jpg")
	if url1 != "http://example.com/foo/bar.jpg" {
		t.Fatal("fail")
	}
}
