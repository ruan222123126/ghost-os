package artifacts

import (
	"strings"
	"testing"
)

func TestNormalizeArtifactIDRejectsInvalidCharacters(t *testing.T) {
	cases := []string{
		"",
		" ",
		".",
		"..",
		"artifact:1",
		"artifact*1",
		"artifact?1",
		"artifact/1",
		`artifact\1`,
		"artifact 1",
		"artifact.1",
		"artifact%2F1",
		"测试",
	}
	for _, value := range cases {
		if _, err := NormalizeArtifactID(value); err == nil {
			t.Fatalf("expected NormalizeArtifactID(%q) to fail", value)
		}
	}
}

func TestNormalizeArtifactIDEnforcesMaxLength(t *testing.T) {
	valid := strings.Repeat("a", MaxIdentifierLength)
	if _, err := NormalizeArtifactID(valid); err != nil {
		t.Fatalf("expected max-length artifact id to be valid: %v", err)
	}

	tooLong := strings.Repeat("a", MaxIdentifierLength+1)
	if _, err := NormalizeArtifactID(tooLong); err == nil {
		t.Fatalf("expected over-length artifact id to fail")
	}
}

func TestNormalizeSessionIDAcceptsUUID(t *testing.T) {
	value := "550e8400-e29b-41d4-a716-446655440000"
	if _, err := NormalizeSessionID(value); err != nil {
		t.Fatalf("expected uuid session id to be valid: %v", err)
	}
}
