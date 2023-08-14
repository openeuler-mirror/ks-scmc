yum install -y golang

go env -w GOPROXY=https://goproxy.io,direct

go get google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

go install google.golang.org/protobuf/cmd/protoc-gen-go
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

BASHRC="$HOME/.bashrc"
echo '' >> $BASHRC
echo '# For golang development' >> $BASHRC
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> $BASHRC
echo '' >> $BASHRC
source $BASHRC