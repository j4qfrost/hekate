// Command hekate is the venue-first AT Protocol CLI client.
//
// Per the salt-mines panel evaluation (2026-04-29) and ADR-tracked roadmap,
// the CLI is the v0.1 reference client — not a fallback. Every protocol
// behaviour the web client supports must also be reachable from here.
//
// Subcommand wiring at v0.1 is stdlib-only (no cobra) so the binary builds
// without external deps. M2 swaps to spf13/cobra and adds the OAuth flow.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const usage = `hekate — venue-first AT Protocol CLI

Usage:
  hekate <command> [args...]

Commands:
  venue create          publish an app.hekate.venue (M2)
  venue list            list known venues (M2; reads server API)
  slot post             publish an app.hekate.slot for a venue (M2)
  slot list             list slots for a venue (M2)
  event claim           claim an open slot by publishing an app.hekate.event (M2)
  rsvp going|maybe|declined   publish an app.hekate.rsvp (M2)
  version               print the build version

Run 'hekate <command> --help' for command-specific flags.

Per ADR 0001 the indigo SDK version is pinned in go.mod; per ADR 0002 the
event/rsvp commands may change shape if Smoke Signal coordination produces
a joint lexicon.`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "hekate:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprintln(stdout, usage)
		return nil
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, "hekate dev (pre-M2)")
		return nil
	case "-h", "--help", "help":
		fmt.Fprintln(stdout, usage)
		return nil
	case "venue", "slot", "event", "rsvp":
		return notImplemented(args[0], args[1:], stderr)
	default:
		fmt.Fprintln(stderr, usage)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func notImplemented(cmd string, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	return fmt.Errorf("%s: not implemented at v0.1; lands with M2", cmd)
}
