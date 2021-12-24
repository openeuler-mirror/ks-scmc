AGENT = ksc-mcube-agent
CONTROLLER = ksc-mcube-controller
RPC = rpc/pb

all: $(AGENT) $(CONTROLLER) $(RPC)

clean:
	rm -f $(AGENT) $(CONTROLLER)

$(RPC):
	cd rpc/ && make

$(AGENT):
	go build -o $@ ksc-mcube/cmd/agent

$(CONTROLLER):
	go build -o $@ ksc-mcube/cmd/controller
