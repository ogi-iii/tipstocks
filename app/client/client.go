package main

import (
	"fmt"
	"log"
	"myTips/tipstocks/app/protobuf"
	"myTips/tipstocks/app/setting"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// Getting the file name & line number if we crashed the go codes
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	opts := grpc.WithInsecure()
	if !setting.Conf.ServerDebug {
		certFile := "ssl/ca.crt"
		creds, sslErr := credentials.NewClientTLSFromFile(certFile, "")
		if sslErr != nil {
			log.Fatalf("Error while loading CA trust certificate: %v", sslErr)
			return
		}
		opts = grpc.WithTransportCredentials(creds)
	}

	target := fmt.Sprintf("localhost:%v", setting.Conf.ServerPort)
	cc, err := grpc.Dial(target, opts)
	if err != nil {
		fmt.Println("could not connect: ", err)
		return
	}
	fmt.Println("Client started!")
	defer cc.Close()

	c := protobuf.NewTipServiceClient(cc)
	fmt.Println(c)
}
