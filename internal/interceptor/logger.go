package interceptor

import (
	"context"
	"log"
	"path"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Logger() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		method := path.Base(info.FullMethod)
		log.Printf("Received request for method: %s", method)

		startTime := time.Now()

		// Call the handler to proceed with the normal execution of the RPC
		resp, err := handler(ctx, req)

		duration := time.Since(startTime)

		if err != nil {
			st, _ := status.FromError(err)
			log.Printf("Request failed for method: %s, error: %v, duration: %v, code: %v", method, st.Err(), duration, st.Code())
		} else {
			log.Printf("Request succeeded for method: %s, duration: %v", method, duration)
		}

		return resp, err
	}
}
