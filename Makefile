PREFIX ?= /usr/local
SBINDIR ?= $(PREFIX)/sbin
SYSCONFDIR ?= $(PREFIX)/etc
SHAREDIR ?= $(PREFIX)/share
MANDIR ?= $(PREFIX)/share/man
SYSTEMDUNITDIR ?= /lib/systemd/system

NAME = nohang
VERSION = 0.3.0
GO ?= go
BUILDFLAGS ?= -ldflags="-s -w"

all: build

build:
	$(GO) build $(BUILDFLAGS) -o bin/$(NAME) ./cmd/$(NAME)/
	$(GO) build $(BUILDFLAGS) -o bin/oom-sort ./cmd/oom-sort/
	$(GO) build $(BUILDFLAGS) -o bin/psi2log ./cmd/psi2log/
	$(GO) build $(BUILDFLAGS) -o bin/psi-top ./cmd/psi-top/

install: base units chcon daemon-reload

base:
	install -d $(DESTDIR)$(SBINDIR)
	install -m 755 bin/$(NAME) $(DESTDIR)$(SBINDIR)/
	install -m 755 bin/oom-sort $(DESTDIR)$(SBINDIR)/
	install -m 755 bin/psi2log $(DESTDIR)$(SBINDIR)/
	install -m 755 bin/psi-top $(DESTDIR)$(SBINDIR)/
	install -d $(DESTDIR)$(SYSCONFDIR)/$(NAME)
	install -m 644 conf/$(NAME)/$(NAME).conf $(DESTDIR)$(SYSCONFDIR)/$(NAME)/
	install -m 644 conf/$(NAME)/$(NAME)-desktop.conf $(DESTDIR)$(SYSCONFDIR)/$(NAME)/
	install -d $(DESTDIR)$(SHAREDIR)/$(NAME)
	echo $(VERSION) > $(DESTDIR)$(SHAREDIR)/$(NAME)/version
	install -d $(DESTDIR)$(MANDIR)/man8
	install -m 644 man/$(NAME).8 $(DESTDIR)$(MANDIR)/man8/
	install -m 644 man/oom-sort.8 $(DESTDIR)$(MANDIR)/man8/
	install -m 644 man/psi2log.8 $(DESTDIR)$(MANDIR)/man8/
	install -m 644 man/psi-top.8 $(DESTDIR)$(MANDIR)/man8/

units:
	install -d $(DESTDIR)$(SYSTEMDUNITDIR)
	sed -e 's|:TARGET_SBINDIR:|$(SBINDIR)|g; s|:TARGET_SYSCONFDIR:|$(SYSCONFDIR)|g' \
	    systemd/$(NAME).service.in > $(DESTDIR)$(SYSTEMDUNITDIR)/$(NAME).service
	sed -e 's|:TARGET_SBINDIR:|$(SBINDIR)|g; s|:TARGET_SYSCONFDIR:|$(SYSCONFDIR)|g' \
	    systemd/$(NAME)-desktop.service.in > $(DESTDIR)$(SYSTEMDUNITDIR)/$(NAME)-desktop.service

chcon:
	-semanage fcontext -a -t bin_t '$(SBINDIR)/$(NAME)' 2>/dev/null || true
	-restorecon -v '$(SBINDIR)/$(NAME)' 2>/dev/null || true

daemon-reload:
	-systemctl daemon-reload 2>/dev/null || true

install-openrc: base
	install -d $(DESTDIR)$(SYSCONFDIR)/init.d
	sed -e 's|:TARGET_SBINDIR:|$(SBINDIR)|g; s|:TARGET_SYSCONFDIR:|$(SYSCONFDIR)|g' \
	    openrc/$(NAME).in > $(DESTDIR)$(SYSCONFDIR)/init.d/$(NAME)
	sed -e 's|:TARGET_SBINDIR:|$(SBINDIR)|g; s|:TARGET_SYSCONFDIR:|$(SYSCONFDIR)|g' \
	    openrc/$(NAME)-desktop.in > $(DESTDIR)$(SYSCONFDIR)/init.d/$(NAME)-desktop

uninstall:
	rm -f $(DESTDIR)$(SBINDIR)/$(NAME)
	rm -f $(DESTDIR)$(SBINDIR)/oom-sort
	rm -f $(DESTDIR)$(SBINDIR)/psi2log
	rm -f $(DESTDIR)$(SBINDIR)/psi-top
	rm -f $(DESTDIR)$(SYSTEMDUNITDIR)/$(NAME).service
	rm -f $(DESTDIR)$(SYSTEMDUNITDIR)/$(NAME)-desktop.service
	rm -rf $(DESTDIR)$(SHAREDIR)/$(NAME)
	rm -f $(DESTDIR)$(MANDIR)/man8/$(NAME).8
	rm -f $(DESTDIR)$(MANDIR)/man8/oom-sort.8
	rm -f $(DESTDIR)$(MANDIR)/man8/psi2log.8
	rm -f $(DESTDIR)$(MANDIR)/man8/psi-top.8

clean:
	rm -rf bin/

.PHONY: all build install base units chcon daemon-reload install-openrc uninstall clean
