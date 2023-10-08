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
	$(PROJECT)-authz.service \

.PHONY: install
install:
	install -m 755 -d $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 -d $(DESTDIR)$(VAR)/lib/$(PROJECT)/{containers,images}
	install -m 755 -d $(DESTDIR)$(VAR)/log/$(PROJECT)
	install -m 644 ./etc/server.toml $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 ./etc/setup_db.sh $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 644 ./etc/database.sql $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 ./etc/graphic_rc $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 ./etc/access-container-gui $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 ./etc/create_network.sh $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 ./etc/sync_image.sh $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 ./etc/setup_agent.sh $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 ./ha/keepalived.sh $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 ./ha/mysql_double_master.sh $(DESTDIR)$(ETC)/$(PROJECT)
	install -m 755 -d $(DESTDIR)$(SYSTEMD)/
	install -m 644 ./systemd/$(PROJECT)-agent.service $(DESTDIR)$(SYSTEMD)/
	install -m 644 ./systemd/$(PROJECT)-controller.service $(DESTDIR)$(SYSTEMD)/
	install -m 644 ./systemd/$(PROJECT)-authz.service $(DESTDIR)$(SYSTEMD)/