/* Copyright (C) 2021 Nicolas Peugnet <n.peugnet@free.fr>

   This file is part of dna-backup.

   dna-backup is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   dna-backup is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with dna-backup.  If not, see <https://www.gnu.org/licenses/>. */

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/n-peugnet/dna-backup/dna"
	"github.com/n-peugnet/dna-backup/logger"
	"github.com/n-peugnet/dna-backup/repo"
)

type command struct {
	Flag  *flag.FlagSet
	Run   func([]string) error
	Usage string
	Help  string
}

const (
	name      = "dna-backup"
	baseUsage = "<command> [<options>] [--] <args>"
)

var (
	logLevel      int
	chunkSize     int
	format        string
	poolCount     int
	trackSize     int
	tracksPerPool int
)

var Commit = command{flag.NewFlagSet("commit", flag.ExitOnError), commitMain,
	"[<options>] [--] <source> <dest>",
	"Create a new version of folder <source> into repo <dest>",
}
var Restore = command{flag.NewFlagSet("restore", flag.ExitOnError), restoreMain,
	"[<options>] [--] <source> <dest>",
	"Restore the last version from repo <source> into folder <dest>",
}
var Export = command{flag.NewFlagSet("export", flag.ExitOnError), exportMain,
	"[<options>] [--] <source> <dest>",
	"Export versions from repo <source> into folder <dest>",
}
var subcommands = map[string]command{
	Commit.Flag.Name():  Commit,
	Restore.Flag.Name(): Restore,
	Export.Flag.Name():  Export,
}

func init() {
	// init default help message
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s %s\n\ncommands:\n", name, baseUsage)
		for _, s := range subcommands {
			fmt.Printf("  %s	%s\n", s.Flag.Name(), s.Help)
		}
		os.Exit(1)
	}
	// setup subcommands
	for _, s := range subcommands {
		s.Flag.IntVar(&logLevel, "v", 3, "log verbosity level (0-4)")
		s.Flag.IntVar(&chunkSize, "c", 8<<10, "chunk size")
	}
	Export.Flag.StringVar(&format, "format", "dir", "format of the export (dir, csv)")
	Export.Flag.IntVar(&poolCount, "pools", 96, "number of pools")
	Export.Flag.IntVar(&trackSize, "track", 1020, "size of a DNA track")
	Export.Flag.IntVar(&tracksPerPool, "tracks-per-pool", 10000, "number of tracks per pool")
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
	}
	cmd, exists := subcommands[args[0]]
	if !exists {
		fmt.Fprintf(flag.CommandLine.Output(), "error: unknown command %s\n\n", args[0])
		flag.Usage()
	}
	cmd.Flag.Usage = func() {
		fmt.Fprintf(cmd.Flag.Output(), "usage: %s %s %s\n\noptions:\n", name, cmd.Flag.Name(), cmd.Usage)
		cmd.Flag.PrintDefaults()
		os.Exit(1)
	}
	cmd.Flag.Parse(args[1:])
	logger.Init(logLevel)
	if err := cmd.Run(cmd.Flag.Args()); err != nil {
		fmt.Fprintf(cmd.Flag.Output(), "error: %s\n\n", err)
		cmd.Flag.Usage()
	}
}

func commitMain(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong number of args")
	}
	source := args[0]
	dest := args[1]
	r := repo.NewRepo(dest, chunkSize)
	r.Commit(source)
	return nil
}

func restoreMain(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong number args")
	}
	source := args[0]
	dest := args[1]
	r := repo.NewRepo(source, chunkSize)
	r.Restore(dest)
	return nil
}

func exportMain(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong number args")
	}
	source := args[0]
	dest := args[1]
	r := repo.NewRepo(source, chunkSize)
	switch format {
	case "dir":
		exporter := dna.New(dest, poolCount, trackSize, tracksPerPool)
		r.Export(exporter)
	case "csv":
		fmt.Println("not yet implemented")
	default:
		logger.Errorf("unknown format %s", format)
	}
	return nil
}
