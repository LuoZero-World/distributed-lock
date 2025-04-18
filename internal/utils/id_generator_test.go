package utils

import (
	"strings"
	"testing"
)

func Test_getLocalIPv4(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{"test1", "192.168", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLocalIPv4()
			if (err != nil) != tt.wantErr {
				t.Errorf("getLocalIPv4() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("getLocalIPv4() = %v, want prefix is %v", got, tt.want)
			}
		})
	}
}
