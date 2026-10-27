package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:10014", "grpc adresi")
	authority := flag.String("authority", "", "sanal host")
	flag.Parse()
	options := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if *authority != "" {
		options = append(options, grpc.WithAuthority(*authority))
	}
	conn, err := grpc.NewClient(*addr, options...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if response.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		fmt.Fprintln(os.Stderr, response.GetStatus())
		os.Exit(1)
	}
	fmt.Println("serving")
}
