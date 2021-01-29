package main

import (
	"context"
	"fmt"
	"log"
	"myTips/tipstocks/app/protobuf"
	"myTips/tipstocks/app/utils"
	"net"
	"os"
	"os/signal"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

func (*server) CreateTip(ctx context.Context, req *protobuf.CreateTipRequest) (*protobuf.CreateTipResponse, error) {
	log.Println("CreateTip requested!")
	tip := req.GetTip()
	data := tipItem{
		Title:       tip.GetTitle(),
		URL:         tip.GetUrl(),
		Description: tip.GetDescription(),
		Image:       tip.GetImage(),
	}
	res, err := collection.InsertOne(ctx, data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v\n", err),
		)
	}
	objID, ok := res.InsertedID.(primitive.ObjectID) // type assertion
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("InsertedID cannot be converted to objID"),
		)
	}
	data.ID = objID
	return &protobuf.CreateTipResponse{Tip: convertDataToTip(&data)}, nil
}

func (*server) DeleteTip(ctx context.Context, req *protobuf.DeleteTipRequest) (*protobuf.DeleteTipResponse, error) {
	log.Println("DeleteTip requested!")
	tipID := req.GetTipId()
	objID, err := primitive.ObjectIDFromHex(tipID) // hex string -> ObjectID
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("cannot parse id: %v\n", err),
		)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.M{"_id": objID} // bson map of filtered id
	res, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("couldn't delete a tip in MongoDB: %v", err),
		)
	} else if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("couldn't delete a tip in MongoDB: %v", err),
		)
	}
	return &protobuf.DeleteTipResponse{
		TipId: tipID,
	}, nil
}

func (*server) AllTips(req *protobuf.AllTipsRequest, stream protobuf.TipService_AllTipsServer) error {
	log.Println("AllTips requested!")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cur, err := findTips(ctx, primitive.D{{}}) // using primitive.D as filter without condition
	defer cur.Close(ctx)
	for cur.Next(ctx) { // cursor iterator
		data := &tipItem{}
		err = cur.Decode(data) // decode cursor to data struct
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("couldn't convert to tip: %v", err),
			)
		}
		stream.Send(&protobuf.AllTipsResponse{
			Tip: convertDataToTip(data),
		})
	}
	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
	return nil
}

func (*server) SearchTips(req *protobuf.SearchTipsRequest, stream protobuf.TipService_SearchTipsServer) error {
	log.Println("SearchTips requested!")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// title filtering: regex with case-insensitive option as "i"
	filter := primitive.D{
		{Key: "title", Value: primitive.Regex{Pattern: req.GetTipTitle(), Options: "i"}},
	}
	cur, err := findTips(ctx, filter)
	defer cur.Close(ctx)
	for cur.Next(ctx) { // cursor iterator
		data := &tipItem{}
		err = cur.Decode(data) // decode cursor to data struct
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("couldn't convert to tip: %v", err),
			)
		}
		stream.Send(&protobuf.SearchTipsResponse{
			Tip: convertDataToTip(data),
		})
	}
	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
	return nil
}

func findTips(ctx context.Context, filter interface{}) (*mongo.Cursor, error) {
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("couldn't find tips from MongoDB: %v", err),
		)
	}
	return cur, nil
}

func convertDataToTip(data *tipItem) *protobuf.Tip {
	return &protobuf.Tip{
		Id:          data.ID.Hex(), // ObjectID -> hex string
		Title:       data.Title,
		Url:         data.URL,
		Description: data.Description,
		Image:       data.Image,
	}
}

// item struct for mongoDB: "bson" means "binary JSON", which is the data format of MongoDB
type tipItem struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"` // can be omitted
	Title       string             `bson:"title"`
	URL         string             `bson:"url"`
	Description string             `bson:"description"`
	Image       string             `bson:"image"`
}

var collection *mongo.Collection // will be used in many functions. (not only main func!)

type server struct {
	protobuf.UnimplementedTipServiceServer // must be contained!
}

func main() {
	// Getting the file name & line number if we crashed the go codes
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf := utils.LoadConf("../utils/config.ini")
	address := fmt.Sprintf("0.0.0.0:%v", conf.ServerPort)
	lis, lisErr := net.Listen("tcp", address)
	if lisErr != nil {
		log.Fatalln("failed to listen: ", lisErr)
		return
	}
	defer fmt.Println("Listener closed.")
	defer lis.Close()

	opts := []grpc.ServerOption{} // blank options
	if !conf.ServerDebug {
		certFile := "../../ssl/server.crt"
		keyFile := "../../ssl/server.pem"
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
	reflection.Register(s) // for Evans (https://github.com/ktr0731/evans)
	fmt.Println("Ready for running server...")

	// Connect to MongoDB: need to be started DB before running server
	dbURI := fmt.Sprintf("mongodb://localhost:%v", conf.DBPort)
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

	collection = client.Database(conf.DBName).Collection(conf.DBCollection)
	fmt.Printf("Connected with MongoDB! (Collection: %v, port: %v)\n", collection.Name(), conf.DBPort)

	// running server as goroutine
	go func() {
		fmt.Printf("Server started! (port: %v)\n", conf.ServerPort)
		if err := s.Serve(lis); err != nil {
			log.Fatalln("failed to serve: ", err)
		}
	}()
	// wait for "Control + C" to exit
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	<-interruptCh // block until receiving an interrupt signal
}
