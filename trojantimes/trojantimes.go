package trojantimes

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/antchfx/xmlquery"

	"github.com/the-rileyj/dsu-info-api/scrapeutils"
)

const (
	MaxConcurrentRequests              = 8
	TrojanTimesArticleSiteMapURL       = "http://trojan-times.com/sitemap-1.xml"
	TrojanTimesArticleImagesSiteMapURL = "http://trojan-times.com/image-sitemap-1.xml"
)

type articleResponse struct {
	url      string
	response *http.Response
}

type Comment struct {
	Body          string    `json:"body"`
	Name          string    `json:"name"`
	DatePublished time.Time `json:"datePublished"`
}

type Article struct {
	Author        string    `json:"author"`
	Body          string    `json:"body"`
	Categories    []string  `json:"categories"`
	Comments      []Comment `json:"comments"`
	DatePublished time.Time `json:"datePublished"`
	Title         string    `json:"Title"`
}

type ArticleImageMetadata struct {
	LastModified time.Time `json:"lastModified"`
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Caption      string    `json:"caption"`
}

type ArticleMap struct {
	articleMap map[string]TrojanTimesArticle
	mutex      *sync.Mutex
}

func (a *ArticleMap) Get(url string) (TrojanTimesArticle, bool) {
	a.mutex.Lock()

	article, exists := a.articleMap[url]

	a.mutex.Unlock()

	return article, exists
}

func (a *ArticleMap) Set(url string, trojanTimesArticle TrojanTimesArticle) {
	a.mutex.Lock()

	a.articleMap[url] = trojanTimesArticle

	a.mutex.Unlock()
}

type TrojanTimesArticle struct {
	Article       Article                `json:"article"`
	ArticleImages []ArticleImageMetadata `json:"articleImages"`
	LastModified  time.Time              `json:"lastModified"`
	Rank          uint                   `json:"rank"`
	URL           string                 `json:"url"`
}

func getTrojanTimesArticlesChannel(articleMap *ArticleMap) chan TrojanTimesArticle {
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
			articleLastUpdated, articleImageLastUpdated   time.Time
			exists                                        bool
			articleImageNode, articleImageMetadataNode    *xmlquery.Node
			trojanTimesArticle                            TrojanTimesArticle
			updateURLs                                    = make(map[string]bool)
			url, articleCaption, articleTitle, articleURL string
		)

		// Adjust to account for when the
		for rank, articleNode := range xmlquery.Find(articleMapDoc, `//url[./loc][./lastmod]`) {
			url = articleNode.SelectElement("loc").InnerText()
			articleLastUpdated, err = time.Parse("2006-01-02T15:04:05Z", articleNode.SelectElement("lastmod").InnerText())

			if err != nil {
				panic(err)
			}

			trojanTimesArticle, exists = articleMap.Get(url)

			updateURLs[url] = !exists || trojanTimesArticle.LastModified != articleLastUpdated

			if !exists {
				trojanTimesArticle = TrojanTimesArticle{
					ArticleImages: make([]ArticleImageMetadata, 0),
					LastModified:  articleLastUpdated,
					Rank:          uint(rank + 1),
					URL:           url,
				}
			} else if trojanTimesArticle.LastModified != articleLastUpdated {
				trojanTimesArticle.LastModified = articleLastUpdated
			}

			for _, articleImageNode = range xmlquery.Find(articleImageMapDoc, fmt.Sprintf(`//url[./loc[text()="%s"]]`, url)) {
				articleImageLastUpdated, err = time.Parse("2006-01-02T15:04:05Z", articleNode.SelectElement("lastmod").InnerText())

				if err != nil {
					panic(err)
				}

				articleImageMetadataNode = articleImageNode.SelectElement("image:image")

				articleCaption = articleImageMetadataNode.SelectElement("image:caption").InnerText()
				articleTitle = articleImageMetadataNode.SelectElement("image:title").InnerText()
				articleURL = articleImageMetadataNode.SelectElement("image:loc").InnerText()

				trojanTimesArticle.ArticleImages = append(
					trojanTimesArticle.ArticleImages,
					ArticleImageMetadata{
						Caption:      articleCaption,
						LastModified: articleImageLastUpdated,
						Title:        articleTitle,
						URL:          articleURL,
					},
				)
			}

			// Update the articleMap just in case the images have changed
			articleMap.Set(url, trojanTimesArticle)

			// Only update the article content that needs to be updated
			if updateURLs[url] {
				articleChan <- trojanTimesArticle
			}
		}

		close(articleChan)
	}()

	return articleChan
}

type articleRequestError struct {
	attempts uint
	errChan  chan error
}

type articleRequestsManager map[string]*articleRequestError

func (aRM articleRequestsManager) getReleaseArticleURLFunc(releaseChan chan bool) func(string) {
	return func(url string) {
		delete(map[string]*articleRequestError(aRM), url)

		<-releaseChan
	}
}

func makeArticleRequests(urlChan chan string) chan articleResponse {
	responseChan := make(chan articleResponse)

	go func() {
		var (
			requestsManager = make(articleRequestsManager)
			fakeErr         = errors.New("")
			err             error
		)

		tokenChan := make(chan bool, MaxConcurrentRequests)
		releaseFunc := requestsManager.getReleaseArticleURLFunc(tokenChan)

		for url := range urlChan {
			tokenChan <- true

			requestErrorHandler := &articleRequestError{}

			// Only send out a request every second
			for err := fakeErr; err != nil; time.Sleep(time.Second) {
				requestErrorHandler.errChan, err = scrapeutils.GetWebpage(url, func(response *http.Response, requestErr error) error {

					return nil
				})
			}

			if err != nil {

			} else {
				releaseFunc(url)
			}
		}
	}()

	return responseChan
}

// func scrapeTrojanTimesArticle() {

// }

// func scrapeTrojanTimesArticlesWithInfo(articleMap *ArticleMap) {
// 	for updateTrojanTimesArticle := range getTrojanTimesArticlesChannelWithInfo(articleMap) {

// 	}
// }
