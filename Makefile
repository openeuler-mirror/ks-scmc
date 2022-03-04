PROJECT = ks-scmc
VERSION = 1.0
TAR_FILE = $(PROJECT)-$(VERSION).tar.gz

.PHONY: backend
backend:
	cd backend && make

.PHONY: client
client:
	cd client && ./build.sh

all: backend client

clean:
	cd backend && make clean && cd -
	cd client && make clean && cd -

install:
	cd backend && make install && cd -
	cd client && make install && cd -

rpm: clean
	rm -f $(TAR_FILE)
	tar -zcf $(TAR_FILE) backend/ client/ rpc_proto/ Makefile
	cp $(TAR_FILE) ~/rpmbuild/SOURCES/
	cp $(PROJECT).spec ~/rpmbuild/SPECS/
	cd ~/rpmbuild/SPECS/
	rpmbuild -ba $(PROJECT).spec
