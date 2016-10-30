DIST = ./dist
GO_VERSION = 1.7.1
SOURCES = $(shell find . \( -path './.git*' -o -path './vendor' \) \
					-prune -o -type f -name '*.go' -print)
XGO_TARGETS := linux/amd64 darwin/amd64 windows/amd64 windows/386
XGO_BUILD_TARGETS := $(foreach t,$(XGO_TARGETS),$(DIST)/$(shell echo "$(t)" \
	| sed 's!\(.*\)/\(.*\)!\2/\1!')/sshclip)
XGO_BUILD_TARGETS := $(foreach t,$(XGO_BUILD_TARGETS), \
	$(shell echo "$(t)" | sed 's!.*windows.*!&.exe!'))

.PHONY: all clean kill-monitor

all: $(DIST) $(XGO_BUILD_TARGETS)

clean: kill-monitor
	rm -rf $(DIST)

kill-monitor:
	@-test -f /tmp/sshclip_monitor.lock \
		&& kill $$(cat /tmp/sshclip_monitor.lock) 2>/dev/null \
		&& rm /tmp/sshclip_monitor.lock

$(XGO_BUILD_TARGETS): kill-monitor $(SOURCES)
	$(eval t := $(wordlist 2,3,$(subst /, ,$@)))
	$(eval target := $(word 2,$(t))/$(word 1,$(t)))
	mkdir -p "$(@D)"
	mkdir -p --mode=2775 "$(@D)/tmp"
	xgo -go $(GO_VERSION) --targets="$(target)" -dest "$(@D)/tmp" ./cmd/sshclip
	cp "$(@D)/tmp/sshclip"* "$@"
	rm -rf "$(@D)/tmp"
	touch "$@"

$(DIST):
	mkdir -p $@
