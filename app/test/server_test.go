package test

import (
	"context"
	"fmt"
	"io"
	"myTips/tipstocks/app/protobuf"
	"myTips/tipstocks/app/utils"
	"myTips/tipstocks/app/utils/goscraper"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

/*
	Before test, you must run server using below commands

	(tipstocks)$ cd app/server
	(tipstocks/app/server)$ go run server.go

	After that, conduct "go test ./..." for all tests
*/
func TestGRPC(t *testing.T) {
	opts := grpc.WithInsecure()
	conf := utils.LoadConf("../utils/config.ini")
	if !conf.ServerDebug {
		certFile := "../../ssl/ca.crt"
		creds, sslErr := credentials.NewClientTLSFromFile(certFile, "")
		if sslErr != nil {
			t.Error("Error while loading CA trust certificate: ", sslErr)
		}
		opts = grpc.WithTransportCredentials(creds)
	}
	target := fmt.Sprintf("localhost:%v", conf.ServerPort)
	cc, err := grpc.Dial(target, opts)
	if err != nil {
		t.Error("could not connect: ", err)
	}
	defer cc.Close()
	c := protobuf.NewTipServiceClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// test gRPC functions
	newTip := createTip(ctx, t, c)
	allTips(ctx, t, c)
	searchTips(ctx, t, c, newTip.GetTitle())
	deleteTip(ctx, t, c, newTip.GetId())
}

func createTip(ctx context.Context, t *testing.T, c protobuf.TipServiceClient) *protobuf.Tip {
	url := "https://github.com/"
	s, err := goscraper.Scrape(url, 5)
	if err != nil {
		t.Error("Cannot get a preview of a webpage: ", err)
	}
	tip := &protobuf.Tip{
		Title:       s.Preview.Title,
		Url:         url,
		Description: s.Preview.Description,
		Image:       s.Preview.Images[0],
	}
	createReq := &protobuf.CreateTipRequest{
		Tip: tip,
	}
	res, err := c.CreateTip(ctx, createReq)
	if err != nil {
		t.Error("Unexpected error: ", err)
	}
	return res.GetTip()
}

func allTips(ctx context.Context, t *testing.T, c protobuf.TipServiceClient) {
	allReq := &protobuf.AllTipsRequest{}
	stream, err := c.AllTips(ctx, allReq)
	if err != nil {
		t.Error("error while calling AllTips: ", err)
	}
	tips := make([]*protobuf.Tip, 0)
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Error("Error happened: ", err)
		}
		tips = append(tips, res.GetTip())
	}
}

func searchTips(ctx context.Context, t *testing.T, c protobuf.TipServiceClient, title string) {
	req := &protobuf.SearchTipsRequest{
		TipTitle: title,
	}
	stream, err := c.SearchTips(ctx, req)
	if err != nil {
		t.Error("error while calling SearchTips: ", err)
	}
	tips := make([]*protobuf.Tip, 0)
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Error("Error happened: ", err)
		}
		tips = append(tips, res.GetTip())
	}
}

func deleteTip(ctx context.Context, t *testing.T, c protobuf.TipServiceClient, id string) {
	req := &protobuf.DeleteTipRequest{
		TipId: id,
	}
	_, err := c.DeleteTip(ctx, req)
	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok {
			if statusErr.Code() == codes.InvalidArgument {
				t.Error("Invalid id received: ", id)
			} else if statusErr.Code() == codes.Internal {
				t.Error("Deletion error in MongoDB")
			} else {
				t.Error("Unexpected error: ", statusErr)
			}
		} else {
			t.Error("error while calling DeleteTip: ", err)
		}
	}
}
