package functionality

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

var (
	ErrActivitiesPostTimeout = errors.New("request to the activities post web page timed out")
)

// type Athletics struct {
// 	Description string  `json:"description"`
// 	Title       string  `json:"title"`
// }

type Contact struct {
	Email string `json:"email"`
	Name  string `json:"who"`
	Phone string `json:"phone"`
}

type Event struct {
	Description string  `json:"description"`
	Title       string  `json:"title"`
	Contact     Contact `json:"contact"`
	When        string  `json:"when"`
	Where       string  `json:"where"`
}

type Employment struct {
	Description string `json:"description"`
	Title       string `json:"title"`
}

type Information struct {
}

type DSUActivities struct {
	Employments  []Employment  `json:"employments"`
	Events       []Event       `json:"events"`
	Informations []Information `json:"informations"`
}

func getEmployments(document *html.Node) []Employment {
	h3Nodes := htmlquery.Find(document, `//fieldset[./legend/h3[contains(text(), "Events")]]/ul/h3`)
	liNodes := htmlquery.Find(document, `//fieldset[./legend/h3[contains(text(), "Events")]]/ul/li`)

	return []Employment{}
}

func getEvents(document *html.Node) []Event {

	return []Event{}
}

func getInformations(document *html.Node) []Information {

	return []Information{}
}

func GetDSUActivitiesInfo() (DSUActivities, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)

	defer cancel()

	return getDSUActivitiesInfoWithClientAndContext(http.DefaultClient, ctx)
}

func getDSUActivitiesInfoWithClientAndContext(client *http.Client, ctx context.Context) (DSUActivities, error) {
	dsuActivitesWebpageRequest, err := http.NewRequest(
		http.MethodGet,
		"https://apps.dsu.edu/DSU-Activities/default.aspx?activities=02/22/2019",
		nil,
	)

	dsuActivitesWebpageRequest = dsuActivitesWebpageRequest.WithContext(ctx)

	errorChannel := make(chan error)

	var (
		dsuActivitesWebpageResponse *http.Response
	)

	go func() {
		var responseErr error

		dsuActivitesWebpageResponse, responseErr = client.Do(dsuActivitesWebpageRequest)

		errorChannel <- responseErr
	}()

	select {
	case <-ctx.Done():
		<-errorChannel

		return DSUActivities{}, ErrActivitiesPostTimeout

	case err := <-errorChannel:
		if err != nil {
			return DSUActivities{}, err
		}
	}

	document, err := htmlquery.Parse(dsuActivitesWebpageResponse.Body)

	if err != nil {
		return DSUActivities{}, err
	}

	return DSUActivities{
		Employments:  getEmployments(document),
		Events:       getEvents(document),
		Informations: getInformations(document),
	}, nil
}

func main() {

}
