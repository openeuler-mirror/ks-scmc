package subscribe

import "testing"

func TestSubscribClient_Subscrib(t *testing.T) {
	type fields struct {
		GRPC   *GrpcInfo
		Server *ServerInfo
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "test01",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			NewSubscribClient().Subscrib()
			// s.Subscrib()
		})
	}
}
