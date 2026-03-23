package pdd

import (
	"encoding/xml"
	"os"
	"testing"
)

func TestConvert(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		golden  string
		tables  int
		fks     int
		indexes int
	}{
		{"Chinook", "testdata/Chinook.pdd", "testdata/Chinook.pgd", 11, 11, 21},
		{"AdventureWorks", "testdata/AdventureWorks.pdd", "testdata/AdventureWorks.pgd", 68, 90, 69},
		{"pagila-light", "testdata/pagila-light.pdd", "testdata/pagila-light.pgd", 15, 22, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.input)
			if err != nil {
				t.Fatal(err)
			}

			project, err := Convert(data, "")
			if err != nil {
				t.Fatal(err)
			}

			got, err := xml.MarshalIndent(project, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = []byte(xml.Header + string(got) + "\n")

			want, err := os.ReadFile(tt.golden)
			if err != nil {
				t.Fatal(err)
			}

			if string(got) != string(want) {
				t.Errorf("output mismatch with golden file %s", tt.golden)
				gotLines := splitLines(string(got))
				wantLines := splitLines(string(want))
				for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
					if gotLines[i] != wantLines[i] {
						t.Errorf("first diff at line %d:\n  got:  %s\n  want: %s", i+1, gotLines[i], wantLines[i])
						break
					}
				}
				if len(gotLines) != len(wantLines) {
					t.Errorf("line count: got %d, want %d", len(gotLines), len(wantLines))
				}
			}

			// count across all schemas
			var tableCount, fkCount, indexCount int
			for _, s := range project.Schemas {
				tableCount += len(s.Tables)
				indexCount += len(s.Indexes)
				for _, tbl := range s.Tables {
					fkCount += len(tbl.FKs)
				}
			}

			if tableCount != tt.tables {
				t.Errorf("tables: got %d, want %d", tableCount, tt.tables)
			}
			if fkCount != tt.fks {
				t.Errorf("fks: got %d, want %d", fkCount, tt.fks)
			}
			if indexCount != tt.indexes {
				t.Errorf("indexes: got %d, want %d", indexCount, tt.indexes)
			}
		})
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
