package whisper

import (
	"log"
	"net"
	"sync"

	empty "github.com/golang/protobuf/ptypes/empty"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

//go:generate protoc -I=$GOPATH/src/github.com/pokstad/whisper --go_out=plugins=grpc:$GOPATH/src $GOPATH/src/github.com/pokstad/whisper/whisper.proto

// Server wraps the core with a grpc interface
type Server struct {
	addr  string
	alias string
	lock  sync.RWMutex      // controls access to peers
	peers map[string]string // maps alias names to network addresses
}

func NewServer(alias, addr string, bootnodes map[string]string) *Server {
	return &Server{
		addr:  addr,
		alias: alias,
		peers: bootnodes,
	}
}

// Serve will begin serving at the indicated address until the context is
// canceled
func (s *Server) Serve(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srvr := grpc.NewServer()
	RegisterWhispererServer(srvr, s)

	done := make(chan struct{})
	defer func() { <-done }() // wait for gorountine to finish

	go func() {
		defer close(done)
		<-ctx.Done()
		srvr.GracefulStop()
	}()

	return srvr.Serve(lis)
}

// Handshake exchanges aliases between this server and a neighbor
func (s *Server) Handshake(ctx context.Context, id *Identity) (*Identity, error) {
	log.Printf("handshake from %#v", id)

	s.lock.Lock()
	defer s.lock.Unlock()

	log.Printf("storing %s for alias %s", id.Addr, id.Alias)
	s.peers[id.Alias] = id.Addr

	return &Identity{
		Alias: s.alias,
		Addr:  s.addr,
	}, nil
}

// Whisper sends a message to a recipient iff recipient is known to server
func (s *Server) Whisper(ctx context.Context, sc *Secret) (*empty.Empty, error) {
	log.Printf("whisper secret %#v", sc)

	// base case: am I the recipient?
	if sc.RecipientAlias == s.alias {
		log.Printf("secret delivered to %s: %s", sc.RecipientAlias, sc.Message)
		return &empty.Empty{}, nil
	}

	// default case: check if recipient is known neighbor
	s.lock.RLock()
	defer s.lock.RUnlock()

	addr, ok := s.peers[sc.RecipientAlias]
	if !ok {
		return &empty.Empty{},
			grpc.Errorf(codes.NotFound, "no route to peer %s", sc.RecipientAlias)
	}

	// address for neighbor is known, attempt to establish connection
	cc, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return nil,
			grpc.Errorf(
				codes.Unavailable,
				"can't dial %s at %s: %s", sc.RecipientAlias, addr, err,
			)
	}

	// neighbor successfully dialed, forward secret
	wc := NewWhispererClient(cc)
	_, err = wc.Whisper(ctx, sc)

	return &empty.Empty{}, err
}
