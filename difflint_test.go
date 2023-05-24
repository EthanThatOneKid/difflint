package difflint

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	// Create a temporary test file
	file, err := os.CreateTemp("", "testfile.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary test file: %v", err)
	}
	defer os.Remove(file.Name())

	// Write test data to the test file
	testData := `//DIFF.IF
DIFF.THEN
`
	_, err = file.WriteString(testData)
	if err != nil {
		t.Fatalf("Failed to write test data to test file: %v", err)
	}
	file.Close()

	// Open the test file for reading
	f, err := os.Open(file.Name())
	if err != nil {
		t.Fatalf("Failed to open test file for reading: %v", err)
	}
	defer f.Close()

	// Call the Parse function
	result, err := Parse(f)
	if err != nil {
		t.Fatalf("Failed to parse test file: %v", err)
	}

	// Assert the expected number of extracted file paths and line number ranges
	if len(result.Paths) != 1 {
		t.Errorf("Expected 1 extracted file path, but got %d", len(result.Paths))
	}
	if len(result.Ranges) != 1 {
		t.Errorf("Expected 1 line number range, but got %d", len(result.Ranges))
	}

	// Assert the expected extracted file path
	expectedPath := "DIFF.THEN"
	if result.Paths[0] != expectedPath {
		t.Errorf("Expected extracted file path to be %q, but got %q", expectedPath, result.Paths[0])
	}

	// Assert the expected line number range
	expectedRange := Range{
		Start: 1,
		End:   1,
	}
	if result.Ranges[0] != expectedRange {
		t.Errorf("Expected line number range to be %+v, but got %+v", expectedRange, result.Ranges[0])
	}
}

func TestParseWithSyntaxError(t *testing.T) {
	// Create a temporary test file
	file, err := os.CreateTemp("", "testfile.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary test file: %v", err)
	}
	defer os.Remove(file.Name())

	// Write test data with a syntax error to the test file
	testData := `//DIFF.IF invalid_text
DIFF.THEN
`
	_, err = file.WriteString(testData)
	if err != nil {
		t.Fatalf("Failed to write test data to test file: %v", err)
	}
	file.Close()

	// Open the test file for reading
	f, err := os.Open(file.Name())
	if err != nil {
		t.Fatalf("Failed to open test file for reading: %v", err)
	}
	defer f.Close()

	// Call the Parse function and expect an error
	_, err = Parse(f)
	if err == nil {
		t.Error("Expected a syntax error, but got no error")
	}
}
