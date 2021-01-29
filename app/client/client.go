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

// ----- echo templates & methods ----- //
type tpl struct {
	templates *template.Template
}

func (t *tpl) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func makeHandler(handler func(c echo.Context, pc protobuf.TipServiceClient) error, pc protobuf.TipServiceClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		render := handler(c, pc)
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

func search(c echo.Context) error {
	data := ""
	return c.Render(http.StatusOK, "search.html", data)
}

func searchResult(c echo.Context, pc protobuf.TipServiceClient) error {
	title := c.FormValue("keywords")
	foundTips, err := searchTips(pc, title)
	if err != nil {
		log.Println(err)
		return c.Redirect(http.StatusFound, "/")
	}
	return c.Render(http.StatusOK, "result.html", foundTips)
}

func blankSearchResult(c echo.Context, pc protobuf.TipServiceClient) error {
	title := ""
	foundTips, err := searchTips(pc, title)
	if err != nil {
		log.Println(err)
		return c.Redirect(http.StatusFound, "/")
	}
	return c.Render(http.StatusOK, "result.html", foundTips)
}

func register(c echo.Context) error {
	err := ""
	return c.Render(http.StatusOK, "register.html", err)
}

func registerNewTip(c echo.Context, pc protobuf.TipServiceClient) error {
	url := c.FormValue("url")
	_, err := createTip(pc, url)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusOK, "register.html", fmt.Sprintln(err))
	}
	return c.Redirect(http.StatusFound, "/")
}

func delete(c echo.Context, pc protobuf.TipServiceClient) error {
	tips, err := allTips(pc)
	if err != nil {
		log.Fatalln(err)
	}
	return c.Render(http.StatusOK, "delete.html", tips)
}

func remove(c echo.Context, pc protobuf.TipServiceClient) error {
	id := c.QueryParam("id")
	err := deleteTip(pc, id)
	if err != nil {
		log.Println(err)
		return c.Redirect(http.StatusFound, "/")
	}
	return c.Redirect(http.StatusFound, "/delete")
}

// ----- gRPC server functions ----- //
func createTip(c protobuf.TipServiceClient, url string) (*protobuf.Tip, error) {
	r, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != 200 {
		return nil, &urlNotFound{url}
	}

	s, err := goscraper.Scrape(url, 5)
	if err != nil {
		log.Println("Cannot get a preview of a webpage: ", err)
		return nil, err
	}
	tip := &protobuf.Tip{
		Title:       s.Preview.Title,
		Url:         url,
		Description: s.Preview.Description,
		Image:       s.Preview.Images[0],
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

type urlNotFound struct {
	url string
}

func (e *urlNotFound) Error() string {
	return fmt.Sprintf("[URL Not Found ...] %v", e.url)
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
		tip := res.GetTip()
		// String is byte array: After converting to word array using []rune, set word counts range as [:num]
		if runeTitle := []rune(tip.GetTitle()); len(runeTitle) > 50 { // string(bytes) -> []rune
			tip.Title = string(runeTitle[:50]) + "…" // []rune -> string
		}
		if runeDescription := []rune(tip.GetDescription()); len(runeDescription) > 150 {
			tip.Description = string([]rune(runeDescription)[:150]) + "…"
		}
		tips = append(tips, tip)
	}
	tips = reverse(tips)
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
	tips = reverse(tips)
	fmt.Println("Tips searched!")
	return tips, nil
}

func reverse(tips []*protobuf.Tip) []*protobuf.Tip {
	for i, j := 0, len(tips)-1; i < j; i, j = i+1, j-1 {
		tips[i], tips[j] = tips[j], tips[i]
	}
	return tips
}

// ----- client funcs ----- //
func main() {
	// Getting the file name & line number if we crashed the go codes
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf := utils.LoadConf("app/utils/config.ini")
	opts := grpc.WithInsecure()
	if !conf.ServerDebug {
		certFile := "app/ssl/ca.crt"
		creds, sslErr := credentials.NewClientTLSFromFile(certFile, "")
		if sslErr != nil {
			log.Fatalf("Error while loading CA trust certificate: %v", sslErr)
		}
		opts = grpc.WithTransportCredentials(creds)
	}
	target := fmt.Sprintf("server:%v", conf.ServerPort)
	cc, err := grpc.Dial(target, opts)
	if err != nil {
		fmt.Println("could not connect: ", err)
	}
	// fmt.Println("Client started!")
	// defer fmt.Println("\nClient stopped.")
	defer cc.Close()
	c := protobuf.NewTipServiceClient(cc)

	e := echo.New()
	t := &tpl{
		templates: template.Must(template.ParseGlob("app/client/src/views/*.html")),
	}
	e.Renderer = t
	e.Static("/css", "app/client/src/css")
	e.Static("/img", "app/client/src/img")
	e.GET("/", makeHandler(index, c))
	e.GET("/search", search)
	e.GET("/search/", search)
	e.GET("/search/result", makeHandler(blankSearchResult, c))
	e.POST("/search/result", makeHandler(searchResult, c))
	e.GET("/register", register)
	e.POST("/register", makeHandler(registerNewTip, c))
	e.GET("/delete", makeHandler(delete, c))
	e.GET("/remove", makeHandler(remove, c))

	// running client as goroutine
	go func() {
		for {
			req := &protobuf.AllTipsRequest{}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_, err := c.AllTips(ctx, req)
			if err != nil {
				continue
			}
			break
		}
		e.Logger.Fatal(e.Start(fmt.Sprintf("0.0.0.0:%v", conf.ClientPort)))
	}()
	// wait for "Control + C" to exit
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	<-interruptCh // block until receiving an interrupt signal
}
