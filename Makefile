BIN := dna-backup
SRC := $(shell find . -not \( -path './exp' -prune \) -type f -name '*.go')
V   := $(if $(CI),-v)

# Default installation paths
PREFIX ?= /usr/local
BINDIR  = $(DESTDIR)$(PREFIX)/bin

.PHONY: all
all: build

.PHONY: build
build: $(BIN)

$(BIN): $(SRC)
	go build $V -o $@

.PHONY: clean
clean:
	rm -rf $(BIN)

.PHONY: test
test:
	go test $V ./... 

.PHONY: exp
exp:
	$(MAKE) -C $@

.PHONY: install
install: $(BIN)
	install -D $(BIN) $(BINDIR)

.PHONY: uninstall
uninstall:
	-rm -f $(BINDIR)/$(BIN)
