package util

import (
	"fmt"
	"testing"
)

func TestDockerCmd(t *testing.T) {
	type args struct {
		id  string
		cmd string
	}
	var tests = []struct {
		name  string
		args  args
		want  string
		want1 int
	}{
		{
			args: args{
				id:  "emqx-emqx-1",
				cmd: "emqx ping",
			},
			want:  "pong",
			want1: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := DockerCmd(tt.args.id, tt.args.cmd)
			println(got)
			println(got1)
			//if got != tt.want {
			//	t.Errorf("DockerCmd() got = %v, want %v", got, tt.want)
			//}
			//if got1 != tt.want1 {
			//	t.Errorf("DockerCmd() got1 = %v, want %v", got1, tt.want1)
			//}
		})
	}
}

func TestCmd(t *testing.T) {
	type args struct {
		cmd string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 int
	}{
		{
			args: args{cmd: "docker ps -a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := Cmd(tt.args.cmd)
			fmt.Println(got)
			fmt.Println(got1)
		})
	}
}
