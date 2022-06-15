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

var githubToken = os.Getenv("TOKEN_GITHUB")

func TestVerifyAlertLinksInSOPs(t TestingTB, ctx *TestingContext) {

	if githubToken == "" {
		t.Skip("github token not present")
	}

	var apiLinks []string
	testUrl := "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/README.md"
	testResp := getSOPAlertLinkStatus(t, testUrl)

	if testResp.StatusCode != 200 {
		t.Log("Response status: ", testUrl, testResp.Status)
	} else {
		t.Log("Response status: ", testResp.Status)
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
						apiLinks = append(apiLinks, string(sopUrl))
					}

				}

			default:
				fmt.Printf("Unknown rule type %s", v)

			}
		}
	}

	apiLinks = unique(apiLinks)
	result := validateSOPLinks(t, apiLinks)
	fmt.Println(result)
}

func modifyLink(rawLink string) (url string) {
	r := strings.NewReplacer(
		"github", "api.github",
		"com/", "com/repos/",
		"blob/master", "contents",
		"tree/master", "contents",
	)

	url = r.Replace(rawLink)
	return url
}

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

func validateSOPLinks(t TestingTB, s []string) bool {
	status := true
	var countFailedLinks int
	for _, url := range s {
		resp := getSOPAlertLinkStatus(t, url)

		if resp.StatusCode != 200 {
			t.Log("Response status :", resp.Status)
			t.Log(url)
			countFailedLinks++
		}

		if countFailedLinks != 0 {
			status = false
		}
	}

	return status

}

func getSOPAlertLinkStatus(t TestingTB, url string) *http.Response {

	url = modifyLink(url)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
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

		}
	}(resp.Body)

	return resp

}
