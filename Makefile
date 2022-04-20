PROJECT = ks-scmc
VERSION = 1.0.0
TAR_FILE = $(PROJECT)-$(VERSION).tar.gz

.PHONY: backend
backend:
	cd backend && make

all: backend

clean:
	cd backend && make clean && cd -

install:
	cd backend && make install && cd -

rpm: clean
	rm -f $(TAR_FILE)
	tar -zcf $(TAR_FILE) backend/ rpc_proto/ Makefile
	cp $(TAR_FILE) ~/rpmbuild/SOURCES/
	cp $(PROJECT).spec ~/rpmbuild/SPECS/
	cd ~/rpmbuild/SPECS/
	rpmbuild -ba $(PROJECT).spec
