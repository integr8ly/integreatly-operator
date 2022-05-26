package common

import (
	"encoding/json"
	"fmt"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"net/http"
	"os"
	"strings"
)

/*
type prometheusAPIResponse struct {
	Status    string                 `json:"status"`
	Data      json.RawMessage        `json:"data"`
	ErrorType prometheusv1.ErrorType `json:"errorType"`
	Error     string                 `json:"error"`
	Warnings  []string               `json:"warnings,omitempty"`
}
*/

func TestVerifyAlertLinksInSPOSs(t TestingTB, ctx *TestingContext) {

	var apiLinks []string

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
				for k, j := range v.Annotations {
					if k == "sop_url" {
						apiLinks = append(apiLinks, string(j))
						url := modifyLinks(apiLinks)
						client := &http.Client{}
						req, err := http.NewRequest("GET", url, nil)
						if err != nil {
							t.Fatalf("Failed to create http request %s", err)
						}

						req.Header.Add("Accept", `application/json`)
						req.Header.Add("Authorization", fmt.Sprintf("token %s", os.Getenv("GITHUB_TOKEN")))
						resp, err := client.Do(req)
						if err != nil {
							t.Fatalf("Failed to make http request %s", err)
						}
						if resp.StatusCode != 200 {
							t.Fatalf("The status code is not 200 %s", err)
						}
						fmt.Printf("%s\n", url)
						fmt.Printf("%d\n", resp.StatusCode)
					}

				}

			default:
				t.Log("Unknown rule type %w", v)

			}
		}
	}

}

func modifyLinks(rawLinks []string) (url string) {
	for _, link := range rawLinks {
		if strings.Contains(link, "blob") {
			url = strings.ReplaceAll(link, "github.com/RHCloudServices/integreatly-help/blob/master", "api.github.com/repos/RHCloudServices/integreatly-help/contents")
		} else if strings.Contains(link, "tree") {
			url = strings.ReplaceAll(link, "github.com/RHCloudServices/integreatly-help/tree/master", "api.github.com/repos/RHCloudServices/integreatly-help/contents")
		}

	}
	return url
}
