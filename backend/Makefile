PROJECT = ks-scmc
BINARY  = ks-scmc-server
RPC     = rpc/pb
DESTDIR :=
PREFIX  := /usr
VAR     := /var
ETC     := /etc

all: $(BINARY)

$(BINARY): $(RPC)
	go build -ldflags="-s -w" scmc/cmd/$@

.PHONY: $(RPC)
$(RPC):
	cd rpc/ && make

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: env
env:
	go env -w GO111MODULE=on
	go env -w GOPROXY=https://goproxy.io,direct
	go get google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
	go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

.PHONY: clean
clean:
	rm -f $(BINARY)
	rm -rf $(RPC)

.PHONY: install
install:
	install -m 755 -D $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)
	cd scripts/ && make install