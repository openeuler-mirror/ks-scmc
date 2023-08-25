package model

import (
	"context"
	"testing"
)

func TestCreateTemplate(t *testing.T) {
	type args struct {
		ctx        context.Context
		id         int64
		name       string
		configbyte []byte
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateTemplate(tt.args.ctx, tt.args.id, tt.args.name, tt.args.configbyte)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreateTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}
