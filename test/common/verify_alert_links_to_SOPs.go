package common

import (
	"encoding/json"
	"fmt"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"io"
	"net/http"
	"os"
	"strings"
)

var githubToken = os.Getenv("GITHUB_TOKEN")

func TestSopUrls(t TestingTB, ctx *TestingContext) {

	if githubToken == "" {
		t.Skip("Github token not provided, use GITHUB_TOKEN environment variable to specify it")
	}

	var sopUrls []string
	testUrl := "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/README.md"
	testResp := getSopUrlStatus(t, testUrl)

	if testResp.StatusCode != 200 {
		t.Fatal("Response status: ", testUrl, testResp.Status, "Given token does not allow access to SOP URLs")
	}

	output, err := execToPod("wget -qO - localhost:9090/api/v1/rules",
		"prometheus-prometheus-0",
		ObservabilityProductNamespace,
		"prometheus", ctx)
	if err != nil {
		t.Fatal("Failed to exec to pod:", err)
	}

	var ApiOutput prometheusAPIResponse

	err = json.Unmarshal([]byte(output), &ApiOutput)
	if err != nil {
		t.Fatal("Failed to unmarshal json", err)
	}

	var rulesResult prometheusv1.RulesResult

	err = json.Unmarshal([]byte(ApiOutput.Data), &rulesResult)
	if err != nil {
		t.Fatal("Failed to unmarshal json", err)
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
	result := validateSopUrls(t, sopUrls)
	fmt.Println(result)
}

func modifySopUrl(sopUrl string) (apiSopUrl string) {
	r := strings.NewReplacer(
		"github", "api.github",
		"com/", "com/repos/",
		"blob/master", "contents",
		"tree/master", "contents",
	)

	apiSopUrl = r.Replace(sopUrl)
	return
}

func unique(sopUrls []string) []string {
	inResult := make(map[string]bool)
	var result []string
	for _, str := range sopUrls {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
			result = append(result, str)
		}
	}
	return result
}

func validateSopUrls(t TestingTB, sopUrls []string) bool {
	status := false
	var countFailedLinks int
	for _, sopUrl := range sopUrls {
		resp := getSopUrlStatus(t, sopUrl)

		if resp.StatusCode != 200 {
			t.Log("Response status :", resp.Status)
			t.Log(sopUrl)
			countFailedLinks++
		}

	}

	if countFailedLinks == 0 {
		status = true
	}

	return status

}

func getSopUrlStatus(t TestingTB, sopUrl string) *http.Response {

	sopUrl = modifySopUrl(sopUrl)
	client := &http.Client{}
	req, err := http.NewRequest("GET", sopurl, nil)
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
			t.Log("Failed to close body")
		}
	}(resp.Body)

	return resp

}
