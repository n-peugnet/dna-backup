BIN := dna-backup
SRC := $(shell find . -not \( -path './exp' -prune \) -type f -name '*.go')
V   := $(if $(CI),-v)
SUBDIRS := exp

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
