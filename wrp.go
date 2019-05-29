package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

var (
	ctx    context.Context
	cancel context.CancelFunc
	scrcap []byte
)

func pageServer(out http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	furl := req.Form["url"]
	var url string
	if len(furl) >= 1 && len(furl[0]) > 4 {
		url = furl[0]
	} else {
		url = "https://www.google.com/"
	}
	log.Printf("%s Page Reqest for %s URL=%s\n", req.RemoteAddr, req.URL.Path, url)
	out.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(out, "<HTML>\n<HEAD><TITLE>WRP %s</TITLE>\n<BODY BGCOLOR=\"#F0F0F0\">", url)
	fmt.Fprintf(out, "<FORM ACTION=\"/\">URL: <INPUT TYPE=\"TEXT\" NAME=\"url\" VALUE=\"%s\">", url)
	fmt.Fprintf(out, "<INPUT TYPE=\"SUBMIT\" VALUE=\"Go\"></FORM><P>\n")
	if len(url) > 4 {
		capture(url, out)
	}
	fmt.Fprintf(out, "</BODY>\n</HTML>\n")
}

func imgServer(out http.ResponseWriter, req *http.Request) {
	log.Printf("%s Img Reqest for %s\n", req.RemoteAddr, req.URL.Path)
	out.Header().Set("Content-Type", "image/png")
	out.Header().Set("Content-Length", strconv.Itoa(len(scrcap)))
	out.Write(scrcap)
}

func capture(url string, out http.ResponseWriter) {
	var nodes []*cdp.Node
	ctxx := chromedp.FromContext(ctx)
	var target string

	log.Printf("Caputure Request for %s\n", url)
	chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(time.Second*2),
		chromedp.CaptureScreenshot(&scrcap),
		chromedp.Nodes("a", &nodes, chromedp.ByQueryAll))

	fmt.Fprintf(out, "<IMG SRC=\"/wrp.png\" ALT=\"wrp\" USEMAP=\"#map\">\n<MAP NAME=\"map\">\n")

	for _, n := range nodes {
		b, err := dom.GetBoxModel().WithNodeID(n.NodeID).Do(cdp.WithExecutor(ctx, ctxx.Target))
		if strings.HasPrefix(n.AttributeValue("href"), "/") {
			target = fmt.Sprintf("/?url=%s%s", url, n.AttributeValue("href"))
		} else {
			target = fmt.Sprintf("/?url=%s", n.AttributeValue("href"))
		}

		if err == nil && len(b.Content) > 6 {
			fmt.Fprintf(out, "<AREA SHAPE=\"RECT\" COORDS=\"%.f,%.f,%.f,%.f\" ALT=\"%s\" TITLE=\"%s\" HREF=\"%s\">\n",
				b.Content[0], b.Content[1], b.Content[4], b.Content[5], n.AttributeValue("href"), n.AttributeValue("href"), target)
		}
	}

	fmt.Fprintf(out, "</MAP>\n")
	log.Printf("Done with caputure for %s\n", url)
}

func main() {
	ctx, cancel = chromedp.NewContext(context.Background())
	defer cancel()
	var addr string
	flag.StringVar(&addr, "l", ":8080", "Listen address:port, default :8080")
	flag.Parse()

	http.HandleFunc("/", pageServer)
	http.HandleFunc("/wrp.png", imgServer)
	log.Printf("Starting http server on %s\n", addr)
	http.ListenAndServe(addr, nil)
}