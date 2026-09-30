package services

import "testing"

func TestGenerateSlug(t *testing.T) {

    slug := GenerateSlug()

    if len(slug) != 6 {
        t.Errorf("expected slug length 6, got %d", len(slug))
    }
}