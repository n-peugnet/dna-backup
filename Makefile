# Copyright (C) 2021 Nicolas Peugnet <n.peugnet@free.fr>

# This file is part of dna-backup.

# dna-backup is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# dna-backup is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with dna-backup.  If not, see <https://www.gnu.org/licenses/>. */

BIN := dna-backup
SRC := $(shell find . -not \( -path './exp' -prune \) -type f -name '*.go')
V   := $(if $(CI),-v)
SUBDIRS := exp pdf

# Default installation paths
PREFIX ?= /usr/local
BINDIR  = $(DESTDIR)$(PREFIX)/bin

.PHONY: all
all: build $(SUBDIRS)

.PHONY: build
build: $(BIN)

$(BIN): $(SRC)
	go build $V -o $@

.PHONY: clean
clean: mostlyclean $(SUBDIRS)

.PHONY: mostlyclean
mostlyclean: $(SUBDIRS)
	rm -rf $(BIN)

.PHONY: test
test:
	go test $V ./...

.PHONY: install
install: $(BIN)
	install -D $(BIN) $(BINDIR)

.PHONY: uninstall
uninstall:
	-rm -f $(BINDIR)/$(BIN)

.PHONY: $(SUBDIRS)
$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)
