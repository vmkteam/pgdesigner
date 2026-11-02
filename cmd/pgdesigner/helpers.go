package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/format"
	"github.com/vmkteam/pgdesigner/pkg/format/pgre"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// saveProject writes a project to a .pgd file.
func saveProject(p *pgd.Project, path string) error {
	data, err := xml.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal XML: %w", err)
	}
	return os.WriteFile(path, []byte(xml.Header+string(data)+"\n"), 0644)
}

// pgdFilePath returns the .pgd output path for a given input, "" for a database.
// For non-.pgd inputs it replaces the extension so we never overwrite the source file.
func pgdFilePath(path string) string {
	if pgre.IsDSN(path) {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".pgd" && ext != "" {
		return strings.TrimSuffix(path, filepath.Ext(path)) + ".pgd"
	}
	return path
}

// parseArgs parses flags placed anywhere in args, so "generate in.pgd -o out.sql"
// works like "generate -o out.sql in.pgd", and returns the positional arguments.
// Arguments after "--" are positional. It exits with usage when the number of
// positional arguments is out of [minArgs, maxArgs]; maxArgs < 0 means no limit.
func parseArgs(fs *flag.FlagSet, args []string, minArgs, maxArgs int) []string {
	var positional []string
	for {
		_ = fs.Parse(args) // flag.ExitOnError: exits on bad flags
		rest := fs.Args()
		if n := len(args) - len(rest); n > 0 && args[n-1] == "--" {
			positional = append(positional, rest...)
			break
		}
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}

	tooMany := maxArgs >= 0 && len(positional) > maxArgs
	if !tooMany && len(positional) >= minArgs {
		return positional
	}
	if tooMany {
		fmt.Fprintf(fs.Output(), "unexpected argument: %s\n\n", positional[maxArgs])
	}
	fs.Usage()
	os.Exit(1)
	return nil
}

// dsnOptions returns database introspection options from the -schema and -full flags.
func dsnOptions(schemas string, full bool) []format.Option {
	opts := []format.Option{format.WithFull(full)}
	if schemas != "" {
		opts = append(opts, format.WithSchemas(strings.Split(schemas, ",")...))
	}
	return opts
}

// writeOutput writes data to path, or to stdout when path is empty.
func writeOutput(path string, data []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}
