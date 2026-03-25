// Command pgdesigner is a PostgreSQL schema designer.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/vmkteam/pgdesigner/frontend"
	"github.com/vmkteam/pgdesigner/pkg/app"
	"github.com/vmkteam/pgdesigner/pkg/designer/store"
	"github.com/vmkteam/pgdesigner/pkg/format/pgre"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

var version = "dev"

func main() {
	// Subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println(version)
			return
		case "convert":
			runConvert(os.Args[2:])
			return
		case "lint":
			runLint(os.Args[2:])
			return
		case "diff":
			runDiff(os.Args[2:])
			return
		case "generate":
			runGenerate(os.Args[2:])
			return
		case "merge":
			runMerge(os.Args[2:])
			return
		case "testdata":
			runTestData(os.Args[2:])
			return
		}
	}

	// Server mode flags
	appMode := flag.Bool("app", false, "open in Chrome/Chromium App Mode (no address bar)")
	port := flag.Int("port", 0, "fixed port (default: random)")
	readOnly := flag.Bool("read-only", false, "read-only mode (disables all write operations)")
	tsClient := flag.Bool("ts_client", false, "generate TypeScript RPC client and exit")
	schemaFilter := flag.String("schema", "", "schema filter for RE (comma-separated)")
	fullRE := flag.Bool("full", false, "full introspection (views, functions, triggers)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `PgDesigner %s — Visual PostgreSQL Schema Designer

Usage:
  pgdesigner [flags] <schema.pgd>    open ERD viewer
  pgdesigner version                 print version and exit
  pgdesigner convert <file>          convert .dbs/.dm2/.pdd/.sql to .pgd
  pgdesigner lint <schema.pgd>       validate schema (use -fix to auto-fix)
  pgdesigner diff <old> <new>        generate ALTER migration SQL
  pgdesigner generate <schema.pgd>   generate DDL SQL
  pgdesigner merge <base> <overlay>  merge two schemas into one .pgd
  pgdesigner testdata <schema.pgd>  generate test data INSERT SQL

Flags:
`, version)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *tsClient {
		p := &pgd.Project{Version: 1, PgVersion: "18"}
		a := app.New(p)
		b, err := a.TypeScriptClient()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(b))
		return
	}

	var st *store.ProjectStore
	var inputPath string

	if flag.NArg() < 1 {
		// Demo mode: start with embedded chinook schema
		st = store.NewProjectStore(&pgd.Project{Version: 1, PgVersion: "18", DefaultSchema: "public", Schemas: []pgd.Schema{{Name: "public"}}}, "")
		st.SetDemo(true)
		inputPath = "(demo)"
	} else {
		inputPath = flag.Arg(0)
		reOpts := pgre.Options{Full: *fullRE}
		if *schemaFilter != "" {
			reOpts.Schemas = strings.Split(*schemaFilter, ",")
		}

		project, err := loadFile(inputPath, reOpts)
		if err != nil {
			log.Fatalf("failed to load %s: %v", inputPath, err)
		}

		st = store.NewProjectStore(project, pgdFilePath(inputPath))
		st.StartAutoBackup(30 * time.Second)
	}

	appOpts := []app.AppOption{app.WithVersion(version)}
	if *readOnly {
		appOpts = append(appOpts, app.WithReadOnly())
	}
	a := app.NewWithStore(st, appOpts...)

	// Track recent file.
	if inputPath != "(demo)" {
		a.Config().AddRecentFile(pgdFilePath(inputPath))
		_ = a.Config().Save()
	}

	if distFS := frontend.DistFS(); distFS != nil {
		a.SetFrontend(distFS)
	}

	addr, err := a.Run(*port)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("PgDesigner serving %s at %s\n", inputPath, addr)
	fmt.Printf("RPC: %s/rpc/  SMDBox: %s/rpc/doc/\n", addr, addr)

	if *appMode {
		openAppMode(addr)
	} else {
		openBrowser(addr)
	}

	fmt.Println("Press Ctrl+C to stop (auto-exits when browser window closes)")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sig:
		fmt.Println("\nStopped.")
	case <-a.QuitCh():
		fmt.Println("\nBrowser closed, exiting.")
	}
}
