package main

import (
	"context"
	"fmt"
	"log"
	"myTips/tipstocks/app/protobuf"
	"net"
	"os"
	"os/signal"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var collection *mongo.Collection // will be used in many functions. (not only main func!)

type server struct {
	protobuf.UnimplementedTipServiceServer // must be contained!
}

// PORT : gRPC server port
const PORT int = 50051

// DEBUG : debug mode without TLS
const DEBUG bool = false

// DBPORT : mongoDB port
const DBPORT int = 27017

func main() {
	// Getting the file name & line number if we crashed the go codes
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	address := fmt.Sprintf("0.0.0.0:%v", PORT)
	lis, lisErr := net.Listen("tcp", address)
	if lisErr != nil {
		log.Fatalln("failed to listen: ", lisErr)
		return
	}
	defer fmt.Println("Listener closed.")
	defer lis.Close()

	opts := []grpc.ServerOption{} // blank options
	if !DEBUG {
		certFile := "ssl/server.crt"
		keyFile := "ssl/server.pem"
		creds, sslErr := credentials.NewServerTLSFromFile(certFile, keyFile)
		if sslErr != nil {
			log.Fatalln("failed to load certificates: ", sslErr)
			return
		}
		opts = append(opts, grpc.Creds(creds))
	}
	s := grpc.NewServer(opts...)
	defer fmt.Println("Server stopped.")
	defer s.Stop()

	protobuf.RegisterTipServiceServer(s, &server{})
	if !DEBUG {
		reflection.Register(s) // for Evans (https://github.com/ktr0731/evans)
	}
	fmt.Println("Ready for running server...")

	// Connect to MongoDB: need to be started DB before running server
	dbURI := fmt.Sprintf("mongodb://localhost:%v", DBPORT)
	client, dbErr := mongo.NewClient(options.Client().ApplyURI(dbURI))
	if dbErr != nil {
		log.Fatalln(dbErr)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dbErr = client.Connect(ctx)
	if dbErr != nil {
		log.Fatalln(dbErr)
		return
	}
	pingErr := client.Ping(ctx, readpref.Primary()) // ping to MongoDB
	if pingErr != nil {
		log.Fatalln("ping error to MongoDB: ", pingErr)
		return
	}
	defer fmt.Println("\nDisconnected with MongoDB.")
	defer client.Disconnect(ctx) // need to be stopped DB after stopping app

	collection = client.Database("tipstocks").Collection("tips") // specify the collection from MongoDB
	fmt.Printf("Connected with MongoDB! (Collection: %v, port: %v)\n", collection.Name(), DBPORT)

	// running server as goroutine
	go func() {
		fmt.Printf("Server started! (port: %v)\n", PORT)
		if err := s.Serve(lis); err != nil {
			log.Fatalln("failed to serve: ", err)
		}
	}()
	// wait for "Control + C" to exit
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	<-interruptCh // block until receiving an interrupt signal
}
