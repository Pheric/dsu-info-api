package trojantimes

import (
	"net/http"
	"time"

	"github.com/antchfx/xmlquery"

	"github.com/the-rileyj/dsu-info-api/scrapeutils"
)

const (
	TrojanTimesArticleSiteMapURL       = "http://trojan-times.com/sitemap-1.xml"
	TrojanTimesArticleImagesSiteMapURL = "http://trojan-times.com/image-sitemap-1.xml"
)

type TrojanTimesArticle struct {
	ImageURLs    []string  `json:"imgUrls"`
	LastModified time.Time `json:"lastModified"`
	Rank         uint      `json:"rank"`
	URL          string    `json:"url"`
}

func GetTrojanTimesArticlesWithInfo(articleMap map[string]TrojanTimesArticle) chan TrojanTimesArticle {
	articleChan := make(chan TrojanTimesArticle)

	go func() {
		var (
			articleMapResponse      *http.Response
			articleImageMapResponse *http.Response
		)

		articleMapResponseChan, err := scrapeutils.GetWebpage(TrojanTimesArticleSiteMapURL, func(articleMapPageResponse *http.Response, err error) error {
			articleMapResponse = articleMapPageResponse

			return err
		})

		if err != nil {
			panic(err)
		}

		articleImageMapResponseChan, err := scrapeutils.GetWebpage(TrojanTimesArticleImagesSiteMapURL, func(articleImageMapPageResponse *http.Response, err error) error {
			articleImageMapResponse = articleImageMapPageResponse

			return err
		})

		if err != nil {
			panic(err)
		}

		var (
			articleMapDoc      *xmlquery.Node
			articleImageMapDoc *xmlquery.Node

			articleMapFinished bool
		)

		select {
		case err = <-articleMapResponseChan:
			articleMapFinished = true

			if err == nil {
				defer articleMapResponse.Body.Close()

				articleMapDoc, err = xmlquery.Parse(articleMapResponse.Body)
			}
		case err = <-articleImageMapResponseChan:
			if err == nil {
				defer articleImageMapResponse.Body.Close()

				articleImageMapDoc, err = xmlquery.Parse(articleImageMapResponse.Body)
			}
		}

		if err != nil {
			panic(err)
		}

		if !articleMapFinished {
			err = <-articleMapResponseChan

			if err == nil {
				defer articleMapResponse.Body.Close()

				articleMapDoc, err = xmlquery.Parse(articleMapResponse.Body)
			}
		} else {
			err = <-articleImageMapResponseChan

			if err == nil {
				defer articleImageMapResponse.Body.Close()

				articleImageMapDoc, err = xmlquery.Parse(articleImageMapResponse.Body)
			}
		}

		var (
			exists             bool
			lastUpdated        time.Time
			trojanTimesArticle TrojanTimesArticle
			updateURLs         map[string]bool
			url                string
		)

		// Adjust to account for when the
		for rank, articleNode := range xmlquery.Find(articleMapDoc, `//url`)[1:] {
			url = articleNode.SelectElement("loc").InnerText()
			lastUpdated, err = time.Parse("2006-01-02T15:04:05Z", articleNode.SelectElement("lastmod").InnerText())

			if err != nil {
				panic(err)
			}

			trojanTimesArticle, exists = articleMap[url]

			if !exists {
				articleMap[url] = TrojanTimesArticle{
					LastModified: lastUpdated,
					Rank:         uint(rank + 1),
					URL:          url,
				}

				updateURLs[url] = true
			} else if trojanTimesArticle.LastModified != lastUpdated {
				trojanTimesArticle.LastModified = lastUpdated

				articleMap[url] = trojanTimesArticle

				updateURLs[url] = true
			}
		}

		// start here tomorrow, keep in mind that when article was updated and when images were updated is different
		for rank, articleNode := range xmlquery.Find(articleImageMapDoc, `//url[./url][./lastmod]`) {
			url = articleNode.SelectElement("loc").InnerText()

			articleMap[url] = TrojanTimesArticle{
				Rank: uint(rank + 1),
				URL:  url,
			}

			articleChan <- articleMap[url]
		}
	}()

	return articleChan
}
