PROJECT = ksc-mcube
RPC = rpc/pb

all: $(PROJECT)

$(PROJECT): $(RPC)
	go build -o $@ $@/cmd

.PHONY: $(RPC)
$(RPC):
	cd rpc/ && make

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: clean
clean:
	rm -f $(PROJECT)
	rm -rf $(RPC)
