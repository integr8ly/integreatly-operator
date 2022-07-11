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

var (
	githubToken   = os.Getenv("GITHUB_TOKEN")
	failedSOPurls = make(chan string)
	wg            sync.WaitGroup
)

func TestSOPUrls(t TestingTB, ctx *TestingContext) {

	if githubToken == "" {
		t.Skip("Github token not provided, use GITHUB_TOKEN environment variable to specify it")
	}

	var sopUrls []string

	// test connection to Github API, with single url

	testUrl := "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/README.md"
	validateGithubToken(t, testUrl)

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

	err = json.Unmarshal(ApiOutput.Data, &rulesResult)
	if err != nil {
		t.Fatal("failed to unmarshal json", err)
	}

	for _, group := range rulesResult.Groups {
		for _, rule := range group.Rules {
			switch v := rule.(type) {
			case prometheusv1.RecordingRule:
			case prometheusv1.AlertingRule:
				for annotation, sopUrl := range v.Annotations {
					if annotation == "sop_url" && sopUrl != "" {
						sopUrls = append(sopUrls, string(sopUrl))
					}
				}

			default:
				t.Log("Unknown rule type %s", v)

			}
		}
	}
	sopUrls = unique(sopUrls)
	validateSOPurls(t, sopUrls)
}

// modify raw link to Github API verison
func convertToGithubApiUrl(sopUrl string) (apiSOPUrl string) {
	r := strings.NewReplacer(
		"github", "api.github",
		"com/", "com/repos/",
		"blob/master", "contents",
		"tree/master", "contents",
	)

	apiSOPUrl = r.Replace(sopUrl)
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

func validateGithubToken(t TestingTB, testUrl string) {
	apiUrl := convertToGithubApiUrl(testUrl)
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
			t.Log("failed to close body")
		}
	}(testResp.Body)

	if testResp.StatusCode != 200 {
		t.Fatal("Response status: ", testUrl, testResp.Status, "Given token does not allow access to SOP URLs")
	}
}

// validate concurrently that links are accessible
func validateSOPurls(t TestingTB, sopUrls []string) {
	for _, url := range sopUrls {
		wg.Add(1)
		go getSOPAlertLinkStatus(t, url, failedSOPurls)

	}

	go func() {
		wg.Wait()
		close(failedSOPurls)
	}()

	if len(failedSOPurls) != 0 {
		for failedSOPUrl := range failedSOPurls {
			t.Log("failed to connect to url: ", failedSOPUrl)

		}
		t.Fatal("test failed due to the invalid url")
	}

}

func getSOPAlertLinkStatus(t TestingTB, url string, failedSOPUrls chan string) {

	defer wg.Done()
	apiUrl := convertToGithubApiUrl(url)
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
	}

}
