package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"myTips/tipstocks/app/protobuf"
	"myTips/tipstocks/app/utils"
	"myTips/tipstocks/app/utils/goscraper"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func main() {
	// Getting the file name & line number if we crashed the go codes
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	opts := grpc.WithInsecure()
	if !utils.Conf.ServerDebug {
		certFile := "ssl/ca.crt"
		creds, sslErr := credentials.NewClientTLSFromFile(certFile, "")
		if sslErr != nil {
			log.Fatalf("Error while loading CA trust certificate: %v", sslErr)
		}
		opts = grpc.WithTransportCredentials(creds)
	}

	target := fmt.Sprintf("localhost:%v", utils.Conf.ServerPort)
	cc, err := grpc.Dial(target, opts)
	if err != nil {
		fmt.Println("could not connect: ", err)
	}
	fmt.Println("Client started!")
	defer cc.Close()

	c := protobuf.NewTipServiceClient(cc)

	// url := "https://www.google.com/"
	// tip, err := createTip(c, url)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// fmt.Println(tip)

	// id := tip.GetId()
	// err = deleteTip(c, id)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// tips, err := allTips(c)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// fmt.Println(tips, len(tips))

	title := "google"
	foundTips, err := searchTips(c, title)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(foundTips, len(foundTips))

}

func createTip(c protobuf.TipServiceClient, url string) (*protobuf.Tip, error) {
	s, err := goscraper.Scrape(url, 5)
	if err != nil {
		log.Println("Cannot get a preview of a webpage: ", err)
		return nil, err
	}
	tip := &protobuf.Tip{
		Title: s.Preview.Title,
		Url:   url,
	}
	req := &protobuf.CreateTipRequest{
		Tip: tip,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := c.CreateTip(ctx, req)
	if err != nil {
		log.Println("Unexpected error: ", err)
		return nil, err
	}
	fmt.Println("New tip created!")
	return res.GetTip(), nil
}

func deleteTip(c protobuf.TipServiceClient, id string) error {
	req := &protobuf.DeleteTipRequest{
		TipId: id,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := c.DeleteTip(ctx, req)
	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok {
			if statusErr.Code() == codes.InvalidArgument {
				log.Println("Invalid id received: ", id)
				return err
			} else if statusErr.Code() == codes.Internal {
				log.Println("Deletion error in MongoDB")
				return err
			} else {
				log.Println("Unexpected error: ", statusErr)
				return err
			}
		} else {
			log.Println("error while calling DeleteTip: ", err)
			return err
		}
	}
	fmt.Println("Delete a tip completed!: ", res.GetTipId())
	return nil
}

func allTips(c protobuf.TipServiceClient) ([]*protobuf.Tip, error) {
	req := &protobuf.AllTipsRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stream, err := c.AllTips(ctx, req)
	if err != nil {
		log.Println("error while calling AllTips: ", err)
		return nil, err
	}
	tips := make([]*protobuf.Tip, 0)
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Error happened: ", err)
			return nil, err
		}
		tips = append(tips, res.GetTip())
	}
	fmt.Println("All tips found!")
	return tips, nil
}

func searchTips(c protobuf.TipServiceClient, title string) ([]*protobuf.Tip, error) {
	req := &protobuf.SearchTipsRequest{
		TipTitle: title,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stream, err := c.SearchTips(ctx, req)
	if err != nil {
		log.Println("error while calling SearchTips: ", err)
		return nil, err
	}
	tips := make([]*protobuf.Tip, 0)
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Error happened: ", err)
			return nil, err
		}
		tips = append(tips, res.GetTip())
	}
	fmt.Println("Tips searched!")
	return tips, nil
}
