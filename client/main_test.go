package main

import (
	"context"
	pb "etcdchat/protobufs"
	"google.golang.org/grpc"
	"testing"
)

func BenchmarkGrcpClient(b *testing.B) {
	conn, err := grpc.Dial("127.0.0.2:8000", grpc.WithInsecure())
	if err != nil {
		b.Errorf("Got error %v", err)
		b.FailNow()
	}

	cli := pb.NewChatNodeClient(conn)

	for i := 0; i < b.N; i++ {
		_, err := cli.SendChatMessage(context.Background(), &pb.ChatMessage{
			Id:      string(i),
			Channel: "foo",
			Message: "Hello foo",
		})
		if err != nil {
			b.Errorf("Got error %v", err)
			b.FailNow()
		}
	}
}
