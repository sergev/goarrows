#
# Arrows game
#
PROG    = goarrows
DESTDIR	= $(HOME)/.local

.PHONY: all install uninstall clean test cover bench gotestsum source

all:
	go build

install: all
	@install -d $(DESTDIR)/usr/bin
	install -m 555 ${PROG} $(DESTDIR)/usr/bin/${PROG}

uninstall:
	rm -f $(DESTDIR)/bin/${PROG}

clean:
	rm -f ${PROG}

#
# For testing, please install gotestsum:
#	go install gotest.tools/gotestsum@latest
#
# Use -timeout 10s so the full suite stays bounded (same as: go test -timeout 10s ./...).
test: gotestsum
	gotestsum --format dots -- -timeout 10s ./...

cover: gotestsum
	gotestsum -- -timeout 10s -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

gotestsum:
	@command -v gotestsum >/dev/null || go install gotest.tools/gotestsum@latest
