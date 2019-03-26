package trojantimes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"

	"github.com/antchfx/xmlquery"
	"github.com/jinzhu/gorm"

	"github.com/the-rileyj/dsu-info-api/scrapeutils"
)

const (
	MaxConcurrentRequests              = 8
	TrojanTimesArticleSiteMapURL       = "http://trojan-times.com/sitemap-1.xml"
	TrojanTimesArticleImagesSiteMapURL = "http://trojan-times.com/image-sitemap-1.xml"
)

var (
	ErrRequestFatal = errors.New("fatal request error, do not retry request")
)

type articleResponse struct {
	document *html.Node
	url      string
}

type TrojanTimesArticle struct {
	Body       ArticleBody      `json:"body" gorm:"not null;foreignkey:ArticleID;association_foreignkey:ID"`
	Images     []ArticleImages  `json:"images" gorm:"not null;foreignkey:ArticleID;association_foreignkey:ID"`
	Comments   []ArticleComment `json:"comments" gorm:"not null;foreignkey:ArticleID;association_foreignkey:ID"`
	Categories []Category       `json:"categories" gorm:"not null;foreignkey:ArticleID;association_foreignkey:ID"` //gorm:"many2many:article_categories"`
	Tags       []Tag            `json:"tags,omitempty" gorm:"not null;foreignkey:ArticleID;association_foreignkey:ID"`

	ID            int       `json:"id" gorm:"primary_key"`
	Rank          uint      `json:"rank"`
	Author        string    `json:"author"`
	Title         string    `json:"title"`
	URL           string    `json:"url" gorm:"not null;unique"`
	DatePublished time.Time `json:"datePublished"`
	LastModified  time.Time `json:"lastModified"`
}

type SkimTrojanTimesArticle struct {
	ID            int       `json:"id" gorm:"primary_key"`
	Rank          uint      `json:"rank"`
	Author        string    `json:"author"`
	Title         string    `json:"title"`
	URL           string    `json:"url" gorm:"not null;unique"`
	DatePublished time.Time `json:"datePublished"`
	LastModified  time.Time `json:"lastModified"`
}

func (t *TrojanTimesArticle) ToSkim() SkimTrojanTimesArticle {
	return SkimTrojanTimesArticle{
		t.ID,
		t.Rank,
		t.Author,
		t.Title,
		t.URL,
		t.DatePublished,
		t.LastModified,
	}
}

type ArticleBody struct {
	ArticleID int `json:"-"`

	Body string `gorm:"not null"`
}

type ArticleImages struct {
	ArticleID int `json:"-"`

	Caption      string    `json:"caption"`
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	LastModified time.Time `json:"lastModified"`
}

type ArticleComment struct {
	ArticleID int `json:"-"`

	Text          string    `json:"text"`
	Name          string    `json:"name"`
	DatePublished time.Time `json:"datePublished"`
}

type Category struct {
	Articles  []*TrojanTimesArticle `gorm:"not null;foreignkey:ID;association_foreignkey:ArticleID"`
	ArticleID int                   `json:"-"`

	ID           uint `gorm:"primary_key"`
	CategoryName string
}

type Tag struct {
	Articles  []*TrojanTimesArticle `gorm:"not null;foreignkey:ID;association_foreignkey:ArticleID"`
	ArticleID int                   `json:"-"`

	ID      uint `gorm:"primary_key"`
	TagName string
}

func getTrojanTimesArticlesChannel() chan TrojanTimesArticle {
	articleChan := make(chan TrojanTimesArticle)

	go func() {
		defer close(articleChan)

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
			articleImageNode, articleImageMetadataNode    *xmlquery.Node
			trojanTimesArticle                            TrojanTimesArticle
			url, articleCaption, articleTitle, articleURL string
		)

		// Adjust to account for when the
		for rank, articleNode := range xmlquery.Find(articleMapDoc, `//url[./loc][./lastmod]`) {
			url = articleNode.SelectElement("loc").InnerText()
			articleLastUpdated, err = time.Parse("2006-01-02T15:04:05Z", articleNode.SelectElement("lastmod").InnerText())

			if err != nil {
				panic(err)
			}

			trojanTimesArticle = TrojanTimesArticle{
				Images:       make([]ArticleImages, 0),
				LastModified: articleLastUpdated,
				Rank:         uint(rank + 1),
				URL:          url,
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

				trojanTimesArticle.Images = append(
					trojanTimesArticle.Images,
					ArticleImages{
						Caption:      articleCaption,
						LastModified: articleImageLastUpdated,
						Title:        articleTitle,
						URL:          articleURL,
					},
				)
			}

			articleChan <- trojanTimesArticle
		}
	}()

	return articleChan
}

type articleRequestManager struct {
	cancelFunc      context.CancelFunc
	responseFunc    func(*http.Response, error) error
	errChan         chan error
	originalRequest *http.Request
}

func makeArticleRequests(urlChan chan string) chan articleResponse {
	responseChan := make(chan articleResponse)

	go func() {
		defer close(responseChan)

		var (
			requestManagers                 = make([]*articleRequestManager, 0)
			originalRequest, contextRequest *http.Request
			err                             error
		)

		tokenChan := make(chan bool, MaxConcurrentRequests)

		finishedAnyRequests := func() bool {
			finishedIndexes := make([]int, 0)

			for finishedIndex, requestManager := range requestManagers {
				select {
				case err = <-requestManager.errChan:
					// Cancel the context to prevent context leaks
					requestManager.cancelFunc()

					if err == nil {
						// Prepend so that we move from back to front when resizing requestManagers
						// preventing the need to recalculate the indexes which need to be removed
						finishedIndexes = append([]int{finishedIndex}, finishedIndexes...)
					} else if err == ErrRequestFatal {
						finishedIndexes = append([]int{finishedIndex}, finishedIndexes...)
					} else {
						ctx, cancelFunc := context.WithTimeout(context.Background(), 7*time.Second)

						contextRequest = requestManager.originalRequest.WithContext(ctx)

						requestManager.cancelFunc = cancelFunc
						requestManager.errChan = scrapeutils.MakeRequestWithContext(ctx, contextRequest, requestManager.responseFunc)
					}
				default:
				}
			}

			for _, finishedIndex := range finishedIndexes {
				requestManagers = append(requestManagers[:finishedIndex], requestManagers[finishedIndex+1:]...)
			}

			return len(finishedIndexes) != 0
		}

		for url := range urlChan {
			tokenChan <- true

			originalRequest, err = http.NewRequest(http.MethodGet, url, nil)

			if err != nil {
				// IDK why making the request would cause error; log once we have a logging solution
				<-tokenChan
				continue
			}

			ctx, cancelFunc := context.WithTimeout(context.Background(), 7*time.Second)

			contextRequest = originalRequest.WithContext(ctx)

			responseFunc := func(curl string) func(*http.Response, error) error {
				return func(response *http.Response, requestErr error) error {
					if requestErr != nil {
						return requestErr
					}

					// Release the token past this point to indicate that the request does not need
					// to be retried either because the request was made successfully or the status
					// code indicated that there is something wrong with requesting to that URL
					<-tokenChan

					if response.StatusCode >= 400 {
						return ErrRequestFatal
					}

					document, requestErr := htmlquery.Parse(response.Body)

					defer response.Body.Close()

					if requestErr != nil {
						return requestErr
					}

					responseChan <- articleResponse{document, curl}

					return nil
				}
			}(url)

			requestManagers = append(requestManagers, &articleRequestManager{
				cancelFunc,
				responseFunc,
				scrapeutils.MakeRequestWithContext(ctx, contextRequest, responseFunc),
				originalRequest,
			})

			if len(requestManagers) == MaxConcurrentRequests {
				for !finishedAnyRequests() {
				}
			} else {
				finishedAnyRequests()
			}
		}

		for len(requestManagers) != 0 {
			finishedAnyRequests()
		}
	}()

	return responseChan
}

func parseTrojanTimesArticle(trojanTimesArticle *html.Node) (TrojanTimesArticle, error) {
	var parsedTrojanTimesArticle TrojanTimesArticle

	// Handle getting the article body
	for _, textNode := range htmlquery.Find(trojanTimesArticle, `//main/article//p`) {
		parsedTrojanTimesArticle.Body.Body += htmlquery.InnerText(textNode)
	}

	// Handle getting the categories for the article
	categoryNodes := htmlquery.Find(trojanTimesArticle, `//footer/span[contains(@class, "cat-links")]//a`)

	parsedTrojanTimesArticle.Categories = make([]Category, 0)

	for _, categoryNode := range categoryNodes {
		parsedTrojanTimesArticle.Categories = append(parsedTrojanTimesArticle.Categories, Category{CategoryName: strings.Trim(htmlquery.InnerText(categoryNode), " ")})
	}

	// Handle getting the tags for the article
	tagsNodes := htmlquery.Find(trojanTimesArticle, `//footer/span[contains(@class, "tags-links")]//a`)

	parsedTrojanTimesArticle.Tags = make([]Tag, 0)

	for _, tagNode := range tagsNodes {
		parsedTrojanTimesArticle.Tags = append(parsedTrojanTimesArticle.Tags, Tag{TagName: strings.Trim(htmlquery.InnerText(tagNode), " ")})
	}

	// Handle getting the author of the article
	authorNode := htmlquery.FindOne(trojanTimesArticle, `//span[contains(@class, "author")]/a/text()`)

	if authorNode == nil {
		return parsedTrojanTimesArticle, errors.New("could not parse the author of the article")
	}

	parsedTrojanTimesArticle.Author = htmlquery.InnerText(authorNode)

	// Handle getting the date the article was published
	datePublishedNode := htmlquery.FindOne(trojanTimesArticle, `//time[contains(@class, "published")]`)

	if datePublishedNode == nil {
		return parsedTrojanTimesArticle, errors.New("could not parse date published, date could not be found")
	}

	var err error

	parsedTrojanTimesArticle.DatePublished, err = time.Parse("2006-01-02T15:04:05-07:00", htmlquery.SelectAttr(datePublishedNode, "datetime"))

	if err != nil {
		return parsedTrojanTimesArticle, errors.New("could not parse date published, time could not be parsed")
	}

	// Handle getting the title of the article
	titleNode := htmlquery.FindOne(trojanTimesArticle, `//h2[contains(@class, "entry-title")]/text()`)

	if titleNode == nil {
		return parsedTrojanTimesArticle, errors.New("could not parse the title of the article")
	}

	parsedTrojanTimesArticle.Title = htmlquery.InnerText(titleNode)

	// Handle getting the comments on the article
	commentNodes := htmlquery.Find(trojanTimesArticle, `//ol[contains(@class, "comment-list")]/li`)

	parsedTrojanTimesArticle.Comments = make([]ArticleComment, 0)

	var (
		articleComment ArticleComment
		itemNode       *html.Node
	)

	for _, commentNode := range commentNodes {
		// Handle getting the comment publish time
		datePublishedNode = htmlquery.FindOne(commentNode, `//time`)

		if datePublishedNode == nil {
			continue
		}

		var err error

		articleComment.DatePublished, err = time.Parse("2006-01-02T15:04:05-07:00", htmlquery.SelectAttr(datePublishedNode, "datetime"))

		if err != nil {
			continue
		}

		// Handle getting the comment author
		itemNode = htmlquery.FindOne(commentNode, `//b`)

		if itemNode == nil {
			continue
		}

		articleComment.Name = htmlquery.InnerText(itemNode)

		// Handle getting the comment text
		itemNode = htmlquery.FindOne(commentNode, `//div[contains(@class, "comment-content")]`)

		if itemNode == nil {
			return parsedTrojanTimesArticle, errors.New("could not parse the text of the comment")
		}

		articleComment.Text = strings.Trim(htmlquery.InnerText(itemNode), " \n\t")

		parsedTrojanTimesArticle.Comments = append(parsedTrojanTimesArticle.Comments, articleComment)
	}

	return parsedTrojanTimesArticle, nil
}

func ScrapeTrojanTimesArticlesWithDatabase(db *gorm.DB) {
	var (
		trojanTimesArticle TrojanTimesArticle
	)

	articleRequestsChan := make(chan string)
	articleResponsesChan := makeArticleRequests(articleRequestsChan)

	go func() {
		// Close down the channel makeArticleRequests is listening on to make it stop listening  for new
		// requests and finish up with existing requests, then shut down it's returned channel, which in
		// turn will signal to other functions in the scraping pipeline to finish their work and return
		defer close(articleRequestsChan)

		var (
			err      error
			oldImage ArticleImages
		)

		for receivedTrojanTimesArticle := range getTrojanTimesArticlesChannel() {
			err = db.First(&trojanTimesArticle, TrojanTimesArticle{URL: receivedTrojanTimesArticle.URL}).Error

			if gorm.IsRecordNotFoundError(err) || (err == nil && trojanTimesArticle.LastModified != receivedTrojanTimesArticle.LastModified) {
				// Establish the Article if it does not exist so that we can update it later
				if gorm.IsRecordNotFoundError(err) {
					db.Create(&receivedTrojanTimesArticle)
				}

				// send to request channel only after creating the article in the database;
				// though extremely unlikely, if we swapped the order, the article could be
				// updated before it is even created
				articleRequestsChan <- receivedTrojanTimesArticle.URL
			} else if err == nil {
				// Check to see if any images have changed
				for _, image := range receivedTrojanTimesArticle.Images {
					err = db.First(&oldImage, ArticleImages{URL: receivedTrojanTimesArticle.URL}).Error

					if gorm.IsRecordNotFoundError(err) {
						db.Create(image)
					} else if err == nil && image.LastModified != oldImage.LastModified {
						db.Model(&oldImage).Updates(&image)
					}
				}
			} else {
				panic(err)
			}
		}
	}()

	for articleRequestResponse := range articleResponsesChan {
		go func(carticleRequestResponse articleResponse) {
			parsedTrojanTimesArticle, err := parseTrojanTimesArticle(carticleRequestResponse.document)

			if err != nil {
				db.Delete(TrojanTimesArticle{URL: carticleRequestResponse.url})
			} else {
				var trojanTimesArticle TrojanTimesArticle

				err = db.First(&trojanTimesArticle, TrojanTimesArticle{URL: carticleRequestResponse.url}).Error

				if err == nil {
					trojanTimesArticle.Author = parsedTrojanTimesArticle.Author
					trojanTimesArticle.Body.Body = parsedTrojanTimesArticle.Body.Body
					trojanTimesArticle.Comments = parsedTrojanTimesArticle.Comments
					trojanTimesArticle.Categories = parsedTrojanTimesArticle.Categories
					trojanTimesArticle.DatePublished = parsedTrojanTimesArticle.DatePublished
					trojanTimesArticle.Tags = parsedTrojanTimesArticle.Tags
					trojanTimesArticle.Title = parsedTrojanTimesArticle.Title

					db.Save(&trojanTimesArticle)
				}
			}
		}(articleRequestResponse)
	}
}
