package app

import (
	"mime"
	"strings"
	"testing"
)

func TestContentDispositionEscapesFilename(t *testing.T) {
	got := contentDisposition("attachment", `workspace/report"evil
.md`)
	if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Fatalf("content disposition contains a newline: %q", got)
	}
	disposition, params, err := mime.ParseMediaType(got)
	if err != nil {
		t.Fatalf("invalid content disposition %q: %v", got, err)
	}
	if disposition != "attachment" {
		t.Fatalf("unexpected disposition: %q", disposition)
	}
	if params["filename"] != "report\"evil\n.md" {
		t.Fatalf("unexpected filename: %q", params["filename"])
	}
}
