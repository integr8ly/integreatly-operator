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

func TestVerifyAlertLinksToSOPs(t TestingTB, ctx *TestingContext) {

	if githubToken == "" {
		t.Skip("github token not present, use TOKEN_GITHUB environment variable to specify it")
	}

	var sopUrls []string
	testUrl := "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/README.md"
	testResp := getSOPAlertLinkStatus(t, testUrl)

	if testResp.StatusCode != 200 {
		t.Fatal("Response status: ", testUrl, testResp.Status, "current given token does not allow access to sop urls")
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
						sopUrls = append(sopUrls, string(sopUrl))
					}

				}

			default:
				fmt.Printf("Unknown rule type %s", v)

			}
		}
	}

	sopUrls = unique(sopUrls)
	result := validateSOPUrls(t, sopUrls)
	fmt.Println(result)
}

func modifySOPUrl(sopUrl string) (apiSOPUrl string) {
	r := strings.NewReplacer(
		"github", "api.github",
		"com/", "com/repos/",
		"blob/master", "contents",
		"tree/master", "contents",
	)

	apiSOPUrl = r.Replace(sopUrl)
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

func validateSOPUrls(t TestingTB, sopUrls []string) bool {
	status := false
	var countFailedLinks int
	for _, sopUrl := range sopUrls {
		resp := getSOPAlertLinkStatus(t, sopUrl)

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

func getSOPAlertLinkStatus(t TestingTB, sopurl string) *http.Response {

	sopurl = modifySOPUrl(sopurl)
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
			t.Log("failed to close body")
		}
	}(resp.Body)

	return resp

}
