package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/pokstad/whisper"
	"google.golang.org/grpc"
)

var (
	// args for both client and server
	alias = flag.String("alias", "", "alias for server")

	// client args
	relay   = flag.String("relay", "", "address of server to relay secret")
	message = flag.String("message", "", "message to relay in whisper")

	// server args
	bindAddr  = flag.String("bind", "", "address to bind server (IP:PORT)")
	bootnodes = flag.String("bootnodes", "", "peer addresses")
)

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatalf("missing command: client or server")
	}

	// last argument is our command
	cmd := os.Args[len(os.Args)-1]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// cancel contexts when interrupt is received
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	go func() {
		<-interrupt
		cancel()
	}()

	switch cmd {

	case "client":
		client(ctx)

	case "server":
		server(ctx)

	default:
		log.Fatalf("unknown command: %s", cmd)

	}
}

func client(ctx context.Context) {
	if *relay == "" {
		log.Fatal("missing relay address")
	}

	if *alias == "" {
		log.Fatal("missing alias")
	}

	cc, err := grpc.DialContext(ctx, *relay, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	wc := whisper.NewWhispererClient(cc)
	_, err = wc.Whisper(ctx, &whisper.Secret{
		RecipientAlias: *alias,
		Message:        *message,
	})
	if err != nil {
		panic(err)
	}
}

func server(ctx context.Context) {
	log.Print("starting whisper server")
	srv := whisper.NewServer(*alias, *bindAddr, getPeers(ctx))
	log.Fatalf("server stopped: %s", srv.Serve(ctx))
}

func getPeers(ctx context.Context) map[string]string {
	addrs := strings.Split(*bootnodes, ",")
	peers := map[string]string{}

	for _, addr := range addrs {
		if addr == "" {
			continue
		}

		log.Printf("dialing bootnode %s", addr)
		cc, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure())
		if err != nil {
			panic(err)
		}

		wc := whisper.NewWhispererClient(cc)
		id, err := wc.Handshake(ctx, &whisper.Identity{
			Alias: *alias,
			Addr:  *bindAddr,
		})
		if err != nil {
			panic(err)
		}

		peers[id.Alias] = addr
	}

	return peers
}
