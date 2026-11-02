package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/designer/diff"
	"github.com/vmkteam/pgdesigner/pkg/designer/gendata"
	"github.com/vmkteam/pgdesigner/pkg/designer/lint"
	"github.com/vmkteam/pgdesigner/pkg/designer/merge"
	"github.com/vmkteam/pgdesigner/pkg/format"
	"github.com/vmkteam/pgdesigner/pkg/format/pgre"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

func runConvert(args []string) {
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	outFile := fs.String("o", "", "output .pgd file (default: input name or database name with .pgd extension)")
	schemaFilter := fs.String("schema", "", "schemas to introspect from a DSN (comma-separated, default: all)")
	fullRE := fs.Bool("full", false, "full introspection from a DSN (views, functions, triggers, types)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner convert [-o output.pgd] [-schema s1,s2] [-full] <input.dbs|.dm2|.pdd|.sql|postgres://...>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	pos := parseArgs(fs, args, 1, 1)

	inputPath := pos[0]
	ext := strings.ToLower(filepath.Ext(inputPath))
	if !pgre.IsDSN(inputPath) && ext == format.ExtPGD {
		log.Fatalf("input is already .pgd: %s", inputPath)
	}

	output, err := convertOutput(inputPath, *outFile)
	if err != nil {
		log.Fatal(err)
	}
	project, err := format.LoadFile(inputPath, dsnOptions(*schemaFilter, *fullRE)...)
	if err != nil {
		log.Fatal(err)
	}
	if err := saveProject(project, output); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Converted %s → %s\n", pgre.RedactDSN(inputPath), output)
}

// convertOutput returns the output file of convert: the -o value, the input with the .pgd
// extension or, for a DSN, <dbname>.pgd in the current directory, which must not exist yet.
func convertOutput(input, output string) (string, error) {
	switch {
	case output != "":
		return output, nil
	case !pgre.IsDSN(input):
		return strings.TrimSuffix(input, filepath.Ext(input)) + format.ExtPGD, nil
	}
	output = filepath.Base(cmp.Or(pgre.DatabaseName(input), "schema")) + format.ExtPGD
	if _, err := os.Stat(output); err == nil {
		return "", fmt.Errorf("%s already exists, set the output file with -o", output)
	}
	return output, nil
}

func runLint(args []string) {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	outputFmt := fs.String("f", "text", "output format: text, json")
	minSeverity := fs.String("s", "info", "minimum severity: error, warning, info")
	fix := fs.Bool("fix", false, "auto-fix fixable issues and save")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner lint [-f text|json] [-s error|warning|info] [-fix] <file.pgd|.dbs|.pdd|.sql>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	pos := parseArgs(fs, args, 1, 1)

	inputPath := pos[0]
	project, err := format.LoadFile(inputPath)
	if err != nil {
		log.Fatal(err)
	}

	issues := lint.Validate(project)

	if *fix {
		issues = lintFix(project, issues, inputPath)
	}

	filtered, errors := filterIssues(issues, parseSeverity(*minSeverity))
	printIssues(filtered, errors, *outputFmt)

	if errors > 0 {
		os.Exit(1)
	}
}

func lintFix(project *pgd.Project, issues []lint.Issue, inputPath string) []lint.Issue {
	var fixable []lint.Issue
	for _, i := range issues {
		if r, ok := lint.Rules[i.Code]; ok && r.Fixable {
			fixable = append(fixable, i)
		}
	}
	if len(fixable) == 0 {
		return issues
	}

	results := lint.Fix(project, fixable)
	fmt.Fprintf(os.Stderr, "Fixed %d issues\n", len(results))

	outPath := pgdFilePath(inputPath)
	if outPath == "" {
		log.Fatal("-fix saves the project to a file: convert the database first")
	}
	if err := saveProject(project, outPath); err != nil {
		log.Fatalf("save %s: %v", outPath, err)
	}
	fmt.Fprintf(os.Stderr, "Saved %s\n", outPath)

	return lint.Validate(project)
}

func parseSeverity(s string) lint.Severity {
	switch strings.ToLower(s) {
	case "error":
		return lint.Error
	case "warning", "warn":
		return lint.Warning
	default:
		return lint.Info
	}
}

func filterIssues(issues []lint.Issue, minSev lint.Severity) (filtered []lint.Issue, errors int) {
	for _, i := range issues {
		if i.Severity <= minSev {
			filtered = append(filtered, i)
			if i.Severity == lint.Error {
				errors++
			}
		}
	}
	return
}

func printIssues(issues []lint.Issue, errors int, outputFmt string) {
	switch outputFmt {
	case "json":
		type jsonIssue struct {
			Severity string `json:"severity"`
			Code     string `json:"code"`
			Title    string `json:"title"`
			Path     string `json:"path"`
			Message  string `json:"message"`
			Fixable  bool   `json:"fixable"`
		}
		out := make([]jsonIssue, len(issues))
		for i, issue := range issues {
			title := issue.Code
			var fixable bool
			if r, ok := lint.Rules[issue.Code]; ok {
				title = r.Title
				fixable = r.Fixable
			}
			out[i] = jsonIssue{
				Severity: issue.Severity.String(),
				Code:     issue.Code,
				Title:    title,
				Path:     issue.Path,
				Message:  issue.Message,
				Fixable:  fixable,
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	default:
		for _, i := range issues {
			fmt.Println(i)
		}
		fmt.Fprintf(os.Stderr, "\n%d errors, %d issues total\n", errors, len(issues))
	}
}

func runDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	outputFmt := fs.String("f", "sql", "output format: sql, json")
	outFile := fs.String("o", "", "output file (default: stdout); skipped when there are no changes")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner diff [-f sql|json] [-o output] <old.pgd> <new.pgd>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	pos := parseArgs(fs, args, 2, 2)

	// a database is introspected fully: without its enums diff would create them again
	oldProject, err := format.LoadFile(pos[0], format.WithFull(true))
	if err != nil {
		log.Fatal(err)
	}
	newProject, err := format.LoadFile(pos[1], format.WithFull(true))
	if err != nil {
		log.Fatal(err)
	}

	result := diff.Diff(oldProject, newProject)
	if len(result.Changes) == 0 {
		fmt.Fprintln(os.Stderr, "no changes")
		return
	}

	payload, err := renderDiff(result, *outputFmt)
	if err != nil {
		log.Fatal(err)
	}
	if err := writeOutput(*outFile, payload); err != nil {
		log.Fatal(err)
	}

	if result.HasHazards() {
		fmt.Fprintln(os.Stderr)
		for _, c := range result.Changes {
			for _, h := range c.Hazards {
				fmt.Fprintf(os.Stderr, "[%s] %s: %s — %s\n", h.Level, h.Code, c.Name, h.Message)
			}
		}
	}
}

func renderDiff(result *diff.DiffResult, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(result.Changes, "", "  ")
	default:
		return []byte(result.SQL()), nil
	}
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	outFile := fs.String("o", "", "output .sql file (default: stdout)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner generate [-o output.sql] <file.pgd|.dbs|.pdd|.sql>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	pos := parseArgs(fs, args, 1, 1)

	project, err := format.LoadFile(pos[0])
	if err != nil {
		log.Fatal(err)
	}

	ddl := pgd.GenerateDDL(project)
	if err := writeOutput(*outFile, []byte(ddl)); err != nil {
		log.Fatal(err)
	}
	if *outFile != "" {
		fmt.Fprintf(os.Stderr, "Generated %s (%d lines)\n", *outFile, strings.Count(ddl, "\n"))
	}
}

func runMerge(args []string) {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	outFile := fs.String("o", "", "output .pgd file (required)")
	layout := fs.String("layout", "both", "layout source: both, base, overlay")
	name := fs.String("name", "", "override project name")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner merge [flags] <base> <overlay> [<extra>...]\n\nMerges multiple schema files into one .pgd. Overlay wins on conflicts.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	pos := parseArgs(fs, args, 2, -1)

	projects := make([]*pgd.Project, len(pos))
	for i, path := range pos {
		p, err := format.LoadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		projects[i] = p
		tables := 0
		for _, s := range p.Schemas {
			tables += len(s.Tables)
		}
		fmt.Fprintf(os.Stderr, "  %s: %d tables\n", filepath.Base(path), tables)
	}

	mergeOpts := merge.Options{Layout: *layout, Name: *name}
	result := projects[0]
	var stats merge.Result
	for i := 1; i < len(projects); i++ {
		result, stats = merge.Merge(result, projects[i], mergeOpts)
	}

	fmt.Fprintf(os.Stderr, "Merge: %d common, %d base-only, %d overlay-only → %d total\n",
		stats.Common, stats.OnlyBase, stats.OnlyOverlay, stats.Total)

	outPath := *outFile
	if outPath == "" {
		outPath = "merged.pgd"
	}
	if err := saveProject(result, outPath); err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "→ %s\n", outPath)
}

func runTestData(args []string) {
	fs := flag.NewFlagSet("testdata", flag.ExitOnError)
	outFile := fs.String("o", "", "output .sql file (default: stdout)")
	seed := fs.Int64("seed", 0, "random seed (0 = random)")
	rows := fs.Int("rows", gendata.DefaultRows, "default rows per table")
	tables := fs.String("tables", "", "generate only these tables (comma-separated)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner testdata [flags] <file.pgd|.dbs|.pdd|.sql>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	pos := parseArgs(fs, args, 1, 1)

	project, err := format.LoadFile(pos[0])
	if err != nil {
		log.Fatal(err)
	}

	opts := gendata.Options{
		Seed: *seed,
		Rows: *rows,
	}

	// parse --tables flag into skip map
	if *tables != "" {
		include := make(map[string]bool)
		for _, t := range strings.Split(*tables, ",") {
			include[strings.TrimSpace(t)] = true
		}
		opts.Tables = make(map[string]gendata.Table)
		for _, s := range project.Schemas {
			for _, t := range s.Tables {
				if !include[t.Name] {
					opts.Tables[t.Name] = gendata.Table{Skip: true}
				}
			}
		}
	}

	var buf strings.Builder
	if err := gendata.Generate(&buf, project, opts); err != nil {
		log.Fatalf("failed to generate test data: %v", err)
	}

	sql := buf.String()
	if err := writeOutput(*outFile, []byte(sql)); err != nil {
		log.Fatal(err)
	}
	if *outFile != "" {
		fmt.Fprintf(os.Stderr, "Generated %s (%d lines)\n", *outFile, strings.Count(sql, "\n"))
	}
}
