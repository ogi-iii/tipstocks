package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"myTips/tipstocks/app/protobuf"
	"myTips/tipstocks/app/utils"
	"myTips/tipstocks/app/utils/goscraper"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type tpl struct {
	templates *template.Template
}

func (t *tpl) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func makeHandler(fn func(c echo.Context, pc protobuf.TipServiceClient) error, pc protobuf.TipServiceClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		render := fn(c, pc) // 各ハンドラーを内部的に実行
		return render
	}
}

func index(c echo.Context, pc protobuf.TipServiceClient) error {
	tips, err := allTips(pc)
	if err != nil {
		log.Fatalln(err)
	}

	return c.Render(http.StatusOK, "index.html", tips)
}

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
	defer fmt.Println("\nClient stopped.")
	defer cc.Close()

	c := protobuf.NewTipServiceClient(cc)
	fmt.Println(c)

	e := echo.New()
	t := &tpl{
		templates: template.Must(template.ParseGlob("app/client/views/*.html")),
	}
	e.Renderer = t
	e.GET("/", makeHandler(index, c))

	// running server as goroutine
	go func() {
		e.Logger.Fatal(e.Start(fmt.Sprintf("0.0.0.0:%v", utils.Conf.ClientPort)))
	}()
	// wait for "Control + C" to exit
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	<-interruptCh // block until receiving an interrupt signal

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

	// title := "google"
	// foundTips, err := searchTips(c, title)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// fmt.Println(foundTips, len(foundTips))

	// uri := "https://www.w3.org/"
	// s, err := goscraper.Scrape(uri, 5)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Printf("Icon : %s\n", s.Preview.Icon)
	// fmt.Printf("Name : %s\n", s.Preview.Name)
	// fmt.Printf("Title : %s\n", s.Preview.Title)
	// fmt.Printf("Description : %s\n", s.Preview.Description)
	// fmt.Printf("Image: %s\n", s.Preview.Images[0])
	// fmt.Printf("Url : %s\n", s.Preview.Link)

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
