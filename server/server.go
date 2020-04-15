package server

import (
	"context"
	"fmt"
	"github.com/autom8ter/geodb/auth"
	"github.com/autom8ter/geodb/config"
	"github.com/autom8ter/geodb/maps"
	"github.com/autom8ter/geodb/stream"
	"github.com/dgraph-io/badger/v2"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/piotrkowalczuk/promgrpc/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"time"
)

type Server struct {
	server     *grpc.Server
	router     *echo.Echo
	streamHub  *stream.Hub
	db         *badger.DB
	hTTPClient *http.Client
	gmaps      *maps.Client
	logger     *log.Logger
}

func (s *Server) GetGRPCServer() *grpc.Server {
	return s.server
}

func (s *Server) GetDB() *badger.DB {
	return s.db
}

func (s *Server) GetStream() *stream.Hub {
	return s.streamHub
}

func (s *Server) GetHTTPClient() *http.Client {
	return s.hTTPClient
}

func (s *Server) GetLogger() *log.Logger {
	return s.logger
}

func (s *Server) GetGmaps() *maps.Client {
	if s.gmaps == nil {
		return nil
	}
	return s.gmaps
}

func GetDeps() (*badger.DB, *stream.Hub, *maps.Client, error) {
	db, err := badger.Open(badger.DefaultOptions(config.Config.GetString("GEODB_PATH")))
	if err != nil {
		return nil, nil, nil, err
	}
	hub := stream.NewHub()
	if config.Config.IsSet("GEODB_GMAPS_KEY") {
		client, err := maps.NewClient(db, config.Config.GetString("GEODB_GMAPS_KEY"), config.Config.GetDuration("GEODB_GMAPS_CACHE_DURATION"))
		if err != nil {
			return db, hub, nil, err
		}
		return db, hub, client, err
	}
	return db, stream.NewHub(), nil, nil
}

func NewServer() (*Server, error) {
	db, hub, gmaps, err := GetDeps()
	if err != nil {
		return nil, err
	}
	var promInterceptor = promgrpc.NewInterceptor(promgrpc.InterceptorOpts{})
	if err := prometheus.DefaultRegisterer.Register(promInterceptor); err != nil {
		return nil, err
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_ctxtags.UnaryServerInterceptor(),
		promInterceptor.UnaryServer(),
		grpc_logrus.UnaryServerInterceptor(log.NewEntry(log.New())),
		grpc_validator.UnaryServerInterceptor(),
		grpc_auth.UnaryServerInterceptor(auth.BasicAuthFunc()),
		grpc_recovery.UnaryServerInterceptor(),
	)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			promInterceptor.StreamServer(),
			grpc_validator.StreamServerInterceptor(),
			grpc_auth.StreamServerInterceptor(auth.BasicAuthFunc()),
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.StatsHandler(promInterceptor),
	)
	s := &Server{
		server:     server,
		router:     echo.New(),
		db:         db,
		hTTPClient: http.DefaultClient,
		logger:     log.New(),
		streamHub:  hub,
		gmaps:      gmaps,
	}
	s.router.Use(
		middleware.Recover(),
	)
	s.router.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	s.hTTPClient.Timeout = 5 * time.Second
	return s, nil
}

func (s *Server) Run() {
	lis, err := net.Listen("tcp", config.Config.GetString("GEODB_PORT"))
	if err != nil {
		s.router.Logger.Fatal(err.Error())
	}
	defer lis.Close()
	defer s.GetDB().Close()

	mux := cmux.New(lis)
	gMux := mux.Match(cmux.HTTP2())
	hMux := mux.Match(cmux.Any())

	fmt.Printf("starting grpc and http server on port %s\n", config.Config.GetString("GEODB_PORT"))

	egp, ctx := errgroup.WithContext(context.Background())
	egp.Go(func() error {
		return s.streamHub.StartObjectStream(ctx)
	})
	egp.Go(func() error {
		for {
			time.Sleep(config.Config.GetDuration("GEODB_GC_INTERVAL"))
			s.db.RunValueLogGC(0.7)
		}
	})
	egp.Go(func() error {
		return s.router.Server.Serve(hMux)
	})
	egp.Go(func() error {
		return s.server.Serve(gMux)
	})
	egp.Go(func() error {
		return mux.Serve()
	})
	if err := egp.Wait(); err != nil {
		s.router.Logger.Fatal(err.Error())
	}
}

func (s *Server) Setup(fn func(s *Server) error) {
	if err := fn(s); err != nil {
		s.GetLogger().Fatal(err.Error())
	}
}
