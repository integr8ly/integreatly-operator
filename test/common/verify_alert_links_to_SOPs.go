package common

import (
	"encoding/json"
	"fmt"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

var(
	githubToken = os.Getenv("TOKEN_GITHUB")
	failedSOPurls = make(chan string)
	wg sync.WaitGroup
	invalidLinksFools = false
)

func TestSOPUrls(t TestingTB, ctx *TestingContext) {

	if githubToken == "" {
		t.Skip("Github token not provided, use GITHUB_TOKEN environment variable to specify it")
	}

	var sopUrls []string

	// test connection to Github API, with single url

	testUrl := "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/README.md"
	apiUrl := modifyLink(testUrl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		log.Fatal("%s", err)
	}

	req.Header.Add("Accept", `application/json`)
	req.Header.Add("Authorization", fmt.Sprintf("token %s", githubToken))
	testResp, err := client.Do(req)
	if err != nil {
		t.Log(err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(testResp.Body)

	if testResp.StatusCode != 200 {
		t.Fatal("Response status: ", testUrl, testResp.Status, "Given token does not allow access to SOP URLs")
	}

	output, err := execToPod("wget -qO - localhost:9090/api/v1/rules",
		"prometheus-prometheus-0",
		ObservabilityProductNamespace,
		"prometheus", ctx)
	if err != nil {
		t.Fatal("failed to exec to pod:", err)
	}

	var ApiOutput prometheusAPIResponse

	err = json.Unmarshal([]byte(output), &ApiOutput)
	if err != nil {
		t.Fatal("failed to unmarshal json", err)
	}

	var rulesResult prometheusv1.RulesResult

	err = json.Unmarshal([]byte(ApiOutput.Data), &rulesResult)
	if err != nil {
		t.Fatal("failed to unmarshal json", err)
	}

	for _, group := range rulesResult.Groups {
		for _, rule := range group.Rules {
			switch v := rule.(type) {
			case prometheusv1.RecordingRule:
			case prometheusv1.AlertingRule:
				for annotation, sopUrl := range v.Annotations {
					if annotation == "sop_url" {
						sopUrls = append(sopUrls, string(sopUrl))
					}
				}

			default:
				fmt.Printf("Unknown rule type %s", v)

			}
		}
	}
	sopUrls = unique(sopUrls)
	validateSOPLinks(t, sopUrls)
}

// modify raw link to Github API verison
func modifyLink(rawLink string) (apiSOPUrl string) {
	r := strings.NewReplacer(
		"github", "api.github",
		"com/", "com/repos/",
		"blob/master", "contents",
		"tree/master", "contents",
	)

	apiSOPUrl = r.Replace(rawLink)
	return
}

// remove duplicate links
func unique(s []string) []string {
	inResult := make(map[string]bool)
	var result []string
	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
			result = append(result, str)
		}
	}
	return result
}

// validate concurrently that links are accessible
func validateSOPLinks(t TestingTB, sopUrls []string) {
	for _, url := range sopUrls {
		wg.Add(1)
		go getSOPAlertLinkStatus(t, url, failedSOPurls)

	}

	go func() {
		wg.Wait()
		close(failedSOPurls)
	}()

	for failedSOPUrl := range failedSOPurls {
		t.Log("failed to connect to url: ", failedSOPUrl)

	}

	if invalidLinksFools {
		t.Fatal("All is lost - links were invalid [intense crying sounds]")
	}
}

func getSOPAlertLinkStatus(t TestingTB, url string, failedSOPUrls chan string) {

	defer wg.Done()
	apiUrl := modifyLink(url)
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		t.Log("%s", err)
	}

	req.Header.Add("Accept", `application/json`)
	req.Header.Add("Authorization", fmt.Sprintf("token %s", githubToken))
	resp, err := client.Do(req)
	if err != nil {
		t.Log(err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Log("failed to close body")
		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		failedSOPUrls <- url
		invalidLinksFools = true
	}

}
