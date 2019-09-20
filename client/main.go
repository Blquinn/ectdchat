package main

import (
	"context"
	pb "etcdchat/protobufs"
	"flag"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"
)

var (
	addr = flag.String("addr", "127.0.0.2:8000", "Server address")
)

func main() {
	conn, err := grpc.Dial("127.0.0.2:8000", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	cli := pb.NewChatNodeClient(conn)
	res, err := cli.SendChatMessage(context.Background(), &pb.ChatMessage{})
	if err != nil {
		panic(err)
	}

	log.Infof("Got hello response %+v", res)
}
