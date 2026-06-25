package qris

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// update controls whether golden files should be (re)generated from
// the current parser output. Run `go test -update` after intentionally
// changing parser behavior to refresh expected outputs.
var update = flag.Bool("update", false, "update golden files")

// TestParseValid_GoldenFiles loops through every payload in
// testdata/valid/, parses it, and compares the JSON-serialized
// Payload against the corresponding golden file in testdata/golden/.
//
// This is the primary regression safety net: if any parser change
// alters output for any of these real-world-shaped payloads, this
// test fails.
func TestParseValid_GoldenFiles(t *testing.T) {
	matches, err := filepath.Glob("testdata/valid/*.txt")
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no testdata/valid/*.txt files found")
	}

	for _, path := range matches {
		path := path // capture for parallel
		name := strings.TrimSuffix(filepath.Base(path), ".txt")

		t.Run(name, func(t *testing.T) {
			payload := readTestFile(t, path)

			got, err := Parse(payload)
			if err != nil {
				t.Fatalf("Parse(%s) failed: %v", name, err)
			}

			gotJSON, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			gotJSON = append(gotJSON, '\n')

			goldenPath := filepath.Join("testdata", "golden", name+".json")

			if *update {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
					t.Fatalf("mkdir golden: %v", err)
				}
				if err := os.WriteFile(goldenPath, gotJSON, 0644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("updated golden: %s", goldenPath)
				return
			}

			wantJSON, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v\n(hint: run `go test -update` to generate)",
					goldenPath, err)
			}

			if !bytes.Equal(gotJSON, wantJSON) {
				t.Errorf("output mismatch for %s\n--- want ---\n%s\n--- got ---\n%s",
					name, wantJSON, gotJSON)
			}
		})
	}
}

// invalidExpectations maps each invalid testdata file to the
// sentinel error that Parse should return for it.
var invalidExpectations = map[string]error{
	"empty.txt":           ErrPayloadTooShort,
	"too_short.txt":       ErrPayloadTooShort,
	"missing_crc_tag.txt": ErrMissingCRCTag,
	"corrupted_crc.txt":   ErrInvalidCRC,
	"invalid_length.txt":  ErrInvalidLength,
	"truncated_tlv.txt":   ErrPayloadTooShort,
}

// TestParseInvalid_ExpectedErrors verifies that each invalid testdata
// file produces the expected sentinel error, ensuring error semantics
// remain stable for library consumers using errors.Is().
func TestParseInvalid_ExpectedErrors(t *testing.T) {
	matches, err := filepath.Glob("testdata/invalid/*.txt")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no testdata/invalid/*.txt files found")
	}

	for _, path := range matches {
		path := path
		name := filepath.Base(path)

		t.Run(name, func(t *testing.T) {
			payload := readTestFile(t, path)

			_, err := Parse(payload)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			want, ok := invalidExpectations[name]
			if !ok {
				t.Fatalf("no expectation defined for %s — add to invalidExpectations map", name)
			}

			if !errors.Is(err, want) {
				t.Errorf("Parse(%s) error = %v; want errors.Is(_, %v) to be true",
					name, err, want)
			}
		})
	}
}

// readTestFile reads a testdata file and trims trailing whitespace
// (including the trailing newline editors often add).
func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}
