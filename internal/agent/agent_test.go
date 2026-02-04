package agent

import (
	"testing"

	"jarvis/internal/types"
)

func TestParseUnifiedDiff(t *testing.T) {
	tests := []struct {
		name      string
		diff      string
		startLine int
		wantLen   int
		wantTypes []types.DiffLineType
	}{
		{
			name:      "empty diff",
			diff:      "",
			startLine: 1,
			wantLen:   0,
			wantTypes: nil,
		},
		{
			name:      "simple addition",
			diff:      "+new line",
			startLine: 1,
			wantLen:   1,
			wantTypes: []types.DiffLineType{types.DiffLineAdded},
		},
		{
			name:      "simple removal",
			diff:      "-old line",
			startLine: 1,
			wantLen:   1,
			wantTypes: []types.DiffLineType{types.DiffLineRemoved},
		},
		{
			name:      "context with changes",
			diff:      " context\n-removed\n+added\n context",
			startLine: 10,
			wantLen:   4,
			wantTypes: []types.DiffLineType{
				types.DiffLineContext,
				types.DiffLineRemoved,
				types.DiffLineAdded,
				types.DiffLineContext,
			},
		},
		{
			name:      "skip hunk headers",
			diff:      "@@ -1,3 +1,4 @@\n context\n+added",
			startLine: 1,
			wantLen:   2,
			wantTypes: []types.DiffLineType{
				types.DiffLineContext,
				types.DiffLineAdded,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUnifiedDiff(tt.diff, tt.startLine)

			if tt.wantLen == 0 {
				if result != nil && len(result) != 0 {
					t.Errorf("parseUnifiedDiff() got %d lines, want 0", len(result))
				}
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("parseUnifiedDiff() got %d lines, want %d", len(result), tt.wantLen)
				return
			}

			for i, wantType := range tt.wantTypes {
				if result[i].Type != wantType {
					t.Errorf("line %d: got type %v, want %v", i, result[i].Type, wantType)
				}
			}
		})
	}
}

func TestParseUnifiedDiffLineNumbers(t *testing.T) {
	diff := " line1\n-line2\n+line2modified\n line3"
	result := parseUnifiedDiff(diff, 5)

	if len(result) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(result))
	}

	if result[0].NewLineNo != 5 {
		t.Errorf("context line: expected NewLineNo 5, got %d", result[0].NewLineNo)
	}

	if result[1].OldLineNo != 6 {
		t.Errorf("removed line: expected OldLineNo 6, got %d", result[1].OldLineNo)
	}

	if result[2].NewLineNo != 6 {
		t.Errorf("added line: expected NewLineNo 6, got %d", result[2].NewLineNo)
	}

	if result[3].NewLineNo != 7 {
		t.Errorf("final context: expected NewLineNo 7, got %d", result[3].NewLineNo)
	}
}

func TestExtractDiffInfo(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := `{"success":true,"file_path":"test.go","unified_diff":"-old\n+new","added_lines":1,"removed_lines":1,"start_line":10}`
		diffInfo := extractDiffInfo(result)

		if diffInfo == nil {
			t.Fatal("expected diffInfo, got nil")
		}

		if diffInfo.FilePath != "test.go" {
			t.Errorf("FilePath: got %q, want %q", diffInfo.FilePath, "test.go")
		}

		if diffInfo.AddedLines != 1 {
			t.Errorf("AddedLines: got %d, want 1", diffInfo.AddedLines)
		}

		if len(diffInfo.Lines) != 2 {
			t.Errorf("Lines: got %d, want 2", len(diffInfo.Lines))
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		result := `invalid json`
		diffInfo := extractDiffInfo(result)

		if diffInfo != nil {
			t.Error("expected nil for invalid json")
		}
	})

	t.Run("unsuccessful result", func(t *testing.T) {
		result := `{"success":false}`
		diffInfo := extractDiffInfo(result)

		if diffInfo != nil {
			t.Error("expected nil for unsuccessful result")
		}
	})
}
