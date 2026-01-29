package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	pb "kucing/pb"

	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	http      = flag.String("http_address", getEnv("HTTP_ADDRESS", "localhost:8888"), "The HTTP server address")
	address   = flag.String("grpc_address", getEnv("GRPC_ADDRESS", "localhost:50051"), "The gRPC server port")
	targetJid = flag.String("target_jid", getEnv("TARGET_JID", ""), "The target JID to send alert")
	deviceJid = flag.String("device_jid", getEnv("DEVICE_JID", ""), "The device JID")
)

func getEnv(key, defaultVal string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultVal
}

func init() {
	flag.Parse()
}

func main() {
	// Connect to gRPC server
	conn, err := grpc.NewClient(
		*address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	// Create client
	client := pb.NewControllerServiceClient(conn)
	fmt.Println("Connected to gRPC server")

	// Example: Check status
	ctx := context.Background()
	resp, err := client.Status(ctx, &pb.StatusRequest{Jid: *deviceJid})
	if err != nil {
		log.Printf("Status call failed: %v", err)
	} else {
		fmt.Printf("Status: %v\n", resp.Status)
	}

	if resp.Status != pb.StatusResponse_STATUS_ACTIVE {
		log.Fatal("Device is not active. Exiting." + resp.Status.String())
	}

	app := echo.New()

	app.POST("/alert", func(c echo.Context) error {
		var req AlertRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(400, map[string]string{"error": "Invalid request"})
		}
		log.Printf("Received alert: %+v", req)

		_, err := client.SendMessage(ctx, &pb.SendMessageRequest{
			Jid:   *deviceJid,
			Phone: *targetJid,
			Body:  req.Message,
		})
		if err != nil {
			log.Printf("SendMessage failed: %v", err)
			return c.JSON(500, map[string]string{"error": "Failed to send message"})
		}

		return c.JSON(200, req)
	})

	app.Start(*http)
}

type AlertRequest struct {
	Heartbeat any    `json:"heartbeat"`
	Monitor   any    `json:"monitor"`
	Message   string `json:"msg"`
}
