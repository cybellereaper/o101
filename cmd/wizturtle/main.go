package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/cybellereaper/wizturtle/v2/internal/patcher"
	"github.com/cybellereaper/wizturtle/v2/internal/state"
)

func main() {
	var (
		patchInfo   = flag.String("patch-info", "", "URL to the patch info JSON document")
		installDir  = flag.String("install-dir", ".", "directory where files will be installed")
		stateFile   = flag.String("state-file", "", "path to the patch state file (defaults to <install-dir>/.wizturtle/state.json)")
		concurrency = flag.Int("concurrency", 0, "number of concurrent downloads (defaults to CPU count)")
		quiet       = flag.Bool("quiet", false, "suppress informational logging")
	)
	flag.Parse()

	if *patchInfo == "" {
		fmt.Fprintln(os.Stderr, "--patch-info is required")
		os.Exit(2)
	}

	if *stateFile == "" {
		*stateFile = filepath.Join(*installDir, ".wizturtle", "state.json")
	}

	var logger patcher.Logger
	if !*quiet {
		std := log.New(os.Stdout, "wizturtle: ", log.LstdFlags)
		logger = std
	}

	store := &state.Store{Path: *stateFile}

	p, err := patcher.New(patcher.Config{
		PatchInfoURL: *patchInfo,
		InstallDir:   *installDir,
		StateStore:   store,
		Concurrency:  *concurrency,
		Logger:       logger,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	start := time.Now()
	if err := p.Run(ctx); err != nil {
		if errors.Is(err, patcher.ErrUpToDate) {
			if !*quiet {
				fmt.Println("Installation is already up to date.")
			}
			return
		}
		fmt.Fprintln(os.Stderr, "patch failed:", err)
		os.Exit(1)
	}

	if !*quiet {
		fmt.Printf("Patch completed in %s\n", time.Since(start).Round(time.Millisecond))
	}
}
