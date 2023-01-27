package validate

import (
	"reflect"
	"testing"
)

func TestParseAllowedRegistries(t *testing.T) {
	type args struct {
		registries string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "parse string",
			args: args{
				registries: "test-1,test-2,test-3",
			},
			want: []string{"test-1", "test-2", "test-3"},
		},
		{
			name: "parse string with whitespaces",
			args: args{
				registries: " \n\n\n  test-1,  \t  test-2, \f\r\vtest-3",
			},
			want: []string{"test-1", "test-2", "test-3"},
		},
		{
			name: "ignore empty element",
			args: args{
				registries: "test-1,,test-2",
			},
			want: []string{"test-1", "test-2"},
		},
		{
			name: "empty string",
			args: args{
				registries: "",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseAllowedRegistries(tt.args.registries); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAllowedRegistries() = %v, want %v", got, tt.want)
			}
		})
	}
}
