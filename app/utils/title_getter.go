package utils

import "github.com/PuerkitoBio/goquery"

// GetURLTitle : Get the title from html
func GetURLTitle(url string) (string, error) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return "", err
	}
	title := doc.Find("title").First()
	return title.Text(), nil
}
