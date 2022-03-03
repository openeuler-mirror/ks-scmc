PROJECT = ks-scmc
PREFIX  := /usr
VAR     := /var
ETC     := /etc
SYSTEMD := $(PREFIX)/lib/systemd/system

all: access-container-gui \
	setup_config.sh \
	database.sql \
	graphic_rc \
	$(PROJECT)-agent.service \
	$(PROJECT)-controller.service \

.PHONY: install
install:
	install -m 755 -d $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 -d $(DESTDIR)$(VAR)/lib/$(PROJECT)/containers
	install -m 755 -d $(DESTDIR)$(VAR)/log/$(PROJECT)
	install -m 755 setup_config.sh $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 644 database.sql $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 graphic_rc $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 access-container-gui $(DESTDIR)$(PREFIX)/bin/
	install -m 755 -d $(DESTDIR)$(SYSTEMD)/
	install -m 644 $(PROJECT)-agent.service $(DESTDIR)$(SYSTEMD)/
	install -m 644 $(PROJECT)-controller.service $(DESTDIR)$(SYSTEMD)/
