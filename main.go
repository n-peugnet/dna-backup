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
	logLevel  int
	chunkSize int
	format    string
)

var commit = command{flag.NewFlagSet("commit", flag.ExitOnError), commitMain,
	"[<options>] [--] <source> <dest>",
	"Create a new version of folder <source> into repo <dest>",
}
var restore = command{flag.NewFlagSet("restore", flag.ExitOnError), restoreMain,
	"[<options>] [--] <source> <dest>",
	"Restore the last version from repo <source> into folder <dest>",
}
var subcommands = map[string]command{
	commit.Flag.Name():  commit,
	restore.Flag.Name(): restore,
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
	repo := repo.NewRepo(dest, chunkSize)
	repo.Commit(source)
	return nil
}

func restoreMain(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("wrong number args")
	}
	source := args[0]
	dest := args[1]
	repo := repo.NewRepo(source, chunkSize)
	repo.Restore(dest)
	return nil
}
