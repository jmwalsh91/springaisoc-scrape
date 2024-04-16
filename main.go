package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"
)

func createHTTPClient() *http.Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	return client
}

func generateURL(page int) string {
	if page == 1 {
		return "https://link.springer.com/search?query=&search-within=Journal&package=openaccessarticles&facet-journal-id=146"
	}
	return fmt.Sprintf("https://link.springer.com/search/page/%d?query=&search-within=Journal&package=openaccessarticles&facet-journal-id=146", page)
}

func findPDFLinks(client *http.Client, pageURL string) ([]string, error) {
	resp, err := client.Get(pageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var links []string
	doc.Find("a.pdf-link").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, ".pdf") {
			fullLink := "https://link.springer.com" + href
			links = append(links, fullLink)
		}
	})

	return links, nil
}

func downloadPDF(client *http.Client, pdfURL, filePath string) error {
	resp, err := client.Get(pdfURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {
	outputDir := flag.String("output", ".", "Directory to save downloaded PDFs")
	startPage := flag.Int("startPage", 1, "The starting page number")
	endPage := flag.Int("endPage", 10, "The ending page number")
	flag.Parse()

	client := createHTTPClient()

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Println("Failed to create output directory:", err)
		return
	}

	for i := *startPage; i <= *endPage; i++ {
		pageURL := generateURL(i)
		fmt.Println("Processing:", pageURL)
		pdfLinks, err := findPDFLinks(client, pageURL)
		if err != nil {
			fmt.Println("Failed to find PDF links for page", i, ":", err)
			continue
		}

		for j, link := range pdfLinks {
			fmt.Println("Downloading PDF from:", link)
			fileName := fmt.Sprintf("journal-page-%d-document-%d.pdf", i, j+1)
			filePath := filepath.Join(*outputDir, fileName)
			err = downloadPDF(client, link, filePath)
			if err != nil {
				fmt.Println("Failed to download PDF from", link, ":", err)
				continue
			}
			fmt.Println("Successfully downloaded:", filePath)
		}
	}
}
