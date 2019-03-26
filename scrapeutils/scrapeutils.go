package scrapeutils

import (
	"context"
	"net/http"
)

// func getWebpageWithContextAndClient() {

// }

// func getWebpageWithContextAndClient(ctx context.Context, request *http.Request, responseFunc func(*http.Response, error) error) chan error {
// 	responseErrorChan := make(chan error, 1)

// 	request = request.WithContext(ctx)

// 	go func() { responseErrorChan <- responseFunc(http.DefaultClient.Do(request)) }()

// 	select {
// 	case <-ctx.Done():
// 		<-responseErrorChan

// 		return ctx.Err()
// 	case err := <-responseErrorChan:
// 		return err
// 	}
// }

func GetWebpage(url string, responseFunc func(*http.Response, error) error) (chan error, error) {
	return GetWebpageWithContext(context.Background(), url, responseFunc)
}

func GetWebpageWithContext(ctx context.Context, url string, responseFunc func(*http.Response, error) error) (chan error, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return nil, err
	}

	return makeWebpageRequestWithContextAndClient(ctx, request, http.DefaultClient, responseFunc), nil
}

func MakeRequestWithContext(ctx context.Context, request *http.Request, responseFunc func(*http.Response, error) error) chan error {
	return makeWebpageRequestWithContextAndClient(ctx, request, http.DefaultClient, responseFunc)
}

func makeWebpageRequestWithContextAndClient(ctx context.Context, request *http.Request, client *http.Client, responseFunc func(*http.Response, error) error) chan error {
	errorChan := make(chan error)

	go func() {
		responseErrorChan := make(chan error)

		go func() { responseErrorChan <- responseFunc(client.Do(request)) }()

		select {
		case <-ctx.Done():
			<-responseErrorChan

			errorChan <- ctx.Err()
		case err := <-responseErrorChan:
			errorChan <- err
		}
	}()

	return errorChan
}
