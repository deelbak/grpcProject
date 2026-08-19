package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/deelbak/grpc/internal/interceptor"
	ufoV1 "github.com/deelbak/grpc/pkg/proto/ufo/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	grpcPort = 50051
	httpPort = 8081
)

type ufoService struct {
	ufoV1.UnimplementedUfoServiceServer

	mu   sync.Mutex
	ufos map[string]*ufoV1.UFO
}

// Create implements the Create method of the UfoServiceServer interface.
func (s *ufoService) Create(_ context.Context, req *ufoV1.CreateRequest) (*ufoV1.CreateResponse, error) {

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf(codes.InvalidArgument.String(), "validation error: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("%d", len(s.ufos)+1)

	ufo := &ufoV1.UFO{
		Id:          id,
		Name:        req.GetName(),
		Description: req.GetDescription(),
	}

	s.ufos[id] = ufo

	return &ufoV1.CreateResponse{
		Ufo: ufo,
	}, nil
}

// Get implements the Get method of the UfoServiceServer interface.
func (s *ufoService) Get(_ context.Context, req *ufoV1.GetRequest) (*ufoV1.GetResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ufo, ok := s.ufos[req.GetId()]
	if !ok {
		return nil, fmt.Errorf("ufo with id %s not found", req.GetId())
	}

	return &ufoV1.GetResponse{
		Ufo: ufo,
	}, nil
}

// Update implements the Update method of the UfoServiceServer interface.
func (s *ufoService) Update(_ context.Context, req *ufoV1.UpdateRequest) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ufo, ok := s.ufos[req.GetId()]
	if !ok {
		return nil, fmt.Errorf("ufo with id %s not found", req.GetId())
	}

	ufo.Name = req.GetName()
	ufo.Description = req.GetDescription()

	return &emptypb.Empty{}, nil
}

// Delete implements the Delete method of the UfoServiceServer interface.
func (s *ufoService) Delete(_ context.Context, req *ufoV1.DeleteRequest) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.ufos[req.GetId()]
	if !ok {
		return nil, fmt.Errorf("ufo with id %s not found", req.GetId())
	}

	delete(s.ufos, req.GetId())

	return &emptypb.Empty{}, nil
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Printf("failed to listen: %v", err)
	}

	defer func() {
		if cerr := lis.Close(); cerr != nil {
			log.Printf("failed to close listener: %v\n", cerr)
		}
	}()

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.Logger()),
	)

	service := &ufoService{
		ufos: make(map[string]*ufoV1.UFO),
	}
	ufoV1.RegisterUfoServiceServer(grpcServer, service)

	reflection.Register(grpcServer)

	go func() {
		log.Printf("starting gRPC server on port %d\n", grpcPort)

		err = grpcServer.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	var gwServer *http.Server

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mux := runtime.NewServeMux()

		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

		err = ufoV1.RegisterUfoServiceHandlerFromEndpoint(
			ctx,
			mux,
			fmt.Sprintf("localhost: %d", grpcPort),
			opts,
		)
		if err != nil {
			log.Fatalf("failed to register gRPC gateway: %v", err)
		}

		fileServer := http.FileServer(http.Dir("./api"))

		httpMux := http.NewServeMux()
		httpMux.Handle("/api/", mux)

		httpMux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", fileServer))
		httpMux.Handle("/swagger.json", fileServer)

		httpMux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.Redirect(w, r, "/swagger-ui/", http.StatusMovedPermanently)
				return
			}
			fileServer.ServeHTTP(w, r)
		}))

		gwServer = &http.Server{
			Addr:              fmt.Sprintf(":%d", httpPort),
			Handler:           httpMux,
			ReadHeaderTimeout: 10 * time.Second,
		}

		log.Printf("starting HTTP server with swagger on port %d\n", httpPort)

		err = gwServer.ListenAndServe()

		if err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to serve HTTP: %v", err)
		}

	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down gRPC server...")

	if gwServer != nil {
		shutDownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := gwServer.Shutdown(shutDownCtx); err != nil {
			log.Printf("failed to shutdown HTTP server: %v", err)
		}

		log.Println("HTTP server stopped")
	}

	lis.Close()

	grpcServer.GracefulStop()
	log.Println("gRPC server stopped")
}
