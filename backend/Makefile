PROJECT = ksc-mcube
RPC = rpc/pb

all: $(PROJECT) $(RPC)

$(PROJECT):
	go build -o $@ $@/cmd

$(RPC):
	cd rpc/ && make

vendor:
	go mod vendor

clean:
	rm -f $(PROJECT)
	rm -rf $(RPC)
