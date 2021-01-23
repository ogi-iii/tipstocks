package main

import (
	"context"
	"fmt"
	"log"
	"myTips/tipstocks/app/protobuf"
	"myTips/tipstocks/app/utils"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	fmt.Println(c)

	url := "http://www.tohoho-web.com/ex/golang.html"
	id, err := createTip(c, url)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(id)
}

func createTip(c protobuf.TipServiceClient, url string) (string, error) {
	title, err := utils.GetURLTitle(url)
	if err != nil {
		log.Println("Cannot get title from url: ", err)
		return "", err
	}
	tip := &protobuf.Tip{
		Title: title,
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
		return "", err
	}
	fmt.Println("New tip created!")
	return res.GetTip().GetId(), nil
}
