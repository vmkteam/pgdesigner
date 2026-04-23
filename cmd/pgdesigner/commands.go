package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	outFile := fs.String("o", "", "output .pgd file (default: input name with .pgd extension)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner convert [-o output.pgd] <input.dbs|.dm2|.pdd|.sql>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	inputPath := fs.Arg(0)
	ext := strings.ToLower(filepath.Ext(inputPath))
	if ext == format.ExtPGD {
		log.Fatalf("input is already .pgd: %s", inputPath)
	}

	project, err := loadFile(inputPath, pgre.Options{})
	if err != nil {
		log.Fatalf("conversion failed: %v", err)
	}

	output := *outFile
	if output == "" {
		output = strings.TrimSuffix(inputPath, ext) + format.ExtPGD
	}
	if err := saveProject(project, output); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Converted %s → %s\n", inputPath, output)
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
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	inputPath := fs.Arg(0)
	project, err := loadFile(inputPath, pgre.Options{})
	if err != nil {
		log.Fatalf("failed to load %s: %v", inputPath, err)
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
	outputFile := fs.String("o", "", "a pathname of an output file. Creates a file if it doesn't exist, overwrites otherwise. Output format is determined by the -f flag")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner diff [-f sql|json] [-o file] <old.pgd> <new.pgd>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fs.Usage()
		os.Exit(1)
	}

	opts := pgre.Options{}
	oldProject, err := loadFile(fs.Arg(0), opts)
	if err != nil {
		log.Fatalf("failed to load %s: %v", fs.Arg(0), err)
	}
	newProject, err := loadFile(fs.Arg(1), opts)
	if err != nil {
		log.Fatalf("failed to load %s: %v", fs.Arg(1), err)
	}

	result := diff.Diff(oldProject, newProject)
	if len(result.Changes) == 0 {
		fmt.Fprintln(os.Stderr, "no changes")
		return
	}

	var w io.Writer = os.Stdout

	if outputFile != nil && *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("failed to open %s: %v", *outputFile, err)
		}
		defer f.Close()
		w = io.MultiWriter(os.Stdout, f)
	}

	switch *outputFmt {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result.Changes); err != nil {
			log.Fatalf("failed to encode JSON: %v", err)
		}
	default:
		if _, err := fmt.Fprint(w, result.SQL()); err != nil {
			log.Fatalf("failed to write output: %v", err)
		}
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

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	outFile := fs.String("o", "", "output .sql file (default: stdout)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pgdesigner generate [-o output.sql] <file.pgd|.dbs|.pdd|.sql>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	project, err := loadFile(fs.Arg(0), pgre.Options{})
	if err != nil {
		log.Fatalf("failed to load %s: %v", fs.Arg(0), err)
	}

	ddl := pgd.GenerateDDL(project)
	if *outFile != "" {
		if err := os.WriteFile(*outFile, []byte(ddl), 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "Generated %s (%d lines)\n", *outFile, strings.Count(ddl, "\n"))
	} else {
		fmt.Print(ddl)
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
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fs.Usage()
		os.Exit(1)
	}

	opts := pgre.Options{}
	projects := make([]*pgd.Project, fs.NArg())
	for i := range fs.NArg() {
		p, err := loadFile(fs.Arg(i), opts)
		if err != nil {
			log.Fatalf("failed to load %s: %v", fs.Arg(i), err)
		}
		projects[i] = p
		tables := 0
		for _, s := range p.Schemas {
			tables += len(s.Tables)
		}
		fmt.Fprintf(os.Stderr, "  %s: %d tables\n", filepath.Base(fs.Arg(i)), tables)
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
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	project, err := loadFile(fs.Arg(0), pgre.Options{})
	if err != nil {
		log.Fatalf("failed to load %s: %v", fs.Arg(0), err)
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
	if *outFile != "" {
		if err := os.WriteFile(*outFile, []byte(sql), 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "Generated %s (%d lines)\n", *outFile, strings.Count(sql, "\n"))
	} else {
		fmt.Print(sql)
	}
}
