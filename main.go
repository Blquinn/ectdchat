package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"sync"
	"time"

	pb "etcdchat/protobufs"
	client "github.com/coreos/etcd/clientv3"
	log "github.com/sirupsen/logrus"
)

var (
	clusterAddr  = flag.String("cluster-addr", "127.0.0.1:4000", "Address of the cluster listener")
	externalAddr = flag.String("external-addr", "127.0.0.1:8000", "Address of the http listener")
)

type chatNodeServer struct{}

func (s *chatNodeServer) SendChatMessage(ctx context.Context, msg *pb.ChatMessage) (*pb.ChatMessageAck, error) {
	return &pb.ChatMessageAck{Id: msg.Id}, nil
}

var _ pb.ChatNodeServer = &chatNodeServer{}

var connectedNodes = struct {
	sync.Mutex

	nodes map[string]bool
}{
	nodes: make(map[string]bool),
}

func addNode(node string) {
	connectedNodes.Lock()
	defer connectedNodes.Unlock()
	connectedNodes.nodes[node] = true
}

func removeNode(node string) {
	connectedNodes.Lock()
	defer connectedNodes.Unlock()
	delete(connectedNodes.nodes, node)
}

func main() {
	flag.Parse()

	cli, err := client.New(client.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	nodeName := *clusterAddr

	res, err := cli.Get(context.Background(), "node", client.WithPrefix())
	if err != nil {
		panic(err)
	}

	for _, kv := range res.Kvs {
		n := string(kv.Value)
		if n != nodeName {
			log.Infof("Found node @ %s", n)
			addNode(n)
		}
	}

	lease, err := cli.Grant(context.Background(), 10)
	if err != nil {
		panic(err)
	}

	_, err = cli.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		panic(err)
	}
	//go func() {
	//	r := <-kaCh
	//	log.Warnf("Keepalive channel returned %+v", r)
	//}()

	_, err = cli.Put(context.Background(), fmt.Sprintf("node:%s", nodeName), nodeName, client.WithLease(lease.ID))
	if err != nil {
		panic(err)
	}

	go func() {
		wc := cli.Watch(context.Background(), "node", client.WithPrefix())
		for {
			v := <-wc
			if v.Err() != nil {
				log.Errorf("Got error while watching nodes: %+v", v.Err())
				break
			}

			for _, e := range v.Events {
				switch e.Type {
				case client.EventTypeDelete:
					log.Infof("Node %s left", e.Kv.Key)
					removeNode(string(e.Kv.Key))
				case client.EventTypePut:
					log.Infof("Node %s joined", e.Kv.Key)
					addNode(string(e.Kv.Key))
				}
			}
		}
	}()

	hub := newHub()
	go hub.run()
	go func() {
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			serveWs(hub, w, r)
		})
		log.Infof("Serving http on %s", *externalAddr)
		err = http.ListenAndServe(*externalAddr, nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}()

	lis, err := net.Listen("tcp", *clusterAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatNodeServer(s, &chatNodeServer{})
	log.Infof("Serving grpc at %s", *clusterAddr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
