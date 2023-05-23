package rpc

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

//listen, _ := net.Listen("tcp", ":9111")
//	server := grpc.NewServer()
//	api.RegisterGoodsApiServer(server, &api.GoodsRpcService{})
//	err := server.Serve(listen)

// VexGrpcServer is a gRPC server
type VexGrpcServer struct {
	listen   net.Listener           // 监听
	g        *grpc.Server           // grpc服务
	register []func(g *grpc.Server) // 注册服务
	ops      []grpc.ServerOption    // grpc服务选项
}

// NewGrpcServer create a gRPC server
func NewGrpcServer(addr string, ops ...VexGrpcOption) (*VexGrpcServer, error) {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	ms := &VexGrpcServer{}
	ms.listen = listen
	for _, v := range ops {
		v.Apply(ms)
	}
	server := grpc.NewServer(ms.ops...)
	ms.g = server
	return ms, nil
}

func (s *VexGrpcServer) Run() error {
	for _, f := range s.register {
		f(s.g)
	}
	return s.g.Serve(s.listen)
}

func (s *VexGrpcServer) Stop() {
	s.g.Stop()
}

func (s *VexGrpcServer) Register(f func(g *grpc.Server)) {
	s.register = append(s.register, f)
}

// VexGrpcOption is a gRPC server option
// this is an interface for VexGrpcServer
type VexGrpcOption interface {
	Apply(s *VexGrpcServer)
}

// DefaultVexGrpcOption is a default gRPC server option
type DefaultVexGrpcOption struct {
	f func(s *VexGrpcServer)
}

// Apply is a method of DefaultVexGrpcOption
func (d *DefaultVexGrpcOption) Apply(s *VexGrpcServer) {
	d.f(s)
}

// WithGrpcOptions is a method of DefaultVexGrpcOption
func WithGrpcOptions(ops ...grpc.ServerOption) VexGrpcOption {
	return &DefaultVexGrpcOption{
		f: func(s *VexGrpcServer) {
			s.ops = append(s.ops, ops...)
		},
	}
}

// VexGrpcClient is a gRPC client
type VexGrpcClient struct {
	Conn *grpc.ClientConn
}

// VexGrpcClientConfig is a gRPC client config
type VexGrpcClientConfig struct {
	Address     string
	Block       bool
	DialTimeout time.Duration
	ReadTimeout time.Duration
	Direct      bool
	KeepAlive   *keepalive.ClientParameters
	dialOptions []grpc.DialOption
}

// DefaultGrpcClientConfig is a default gRPC client config
func DefaultGrpcClientConfig() *VexGrpcClientConfig {
	return &VexGrpcClientConfig{
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		DialTimeout: time.Second * 3,
		ReadTimeout: time.Second * 2,
		Block:       true,
	}
}

// NewGrpcClient create a gRPC client
func NewGrpcClient(config *VexGrpcClientConfig) (*VexGrpcClient, error) {
	var ctx = context.Background()
	var dialOptions = config.dialOptions

	if config.Block {
		//阻塞
		if config.DialTimeout > time.Duration(0) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.DialTimeout)
			defer cancel()
		}
		dialOptions = append(dialOptions, grpc.WithBlock())
	}
	if config.KeepAlive != nil {
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(*config.KeepAlive))
	}
	conn, err := grpc.DialContext(ctx, config.Address, dialOptions...)
	if err != nil {
		return nil, err
	}
	return &VexGrpcClient{
		Conn: conn,
	}, nil
}
