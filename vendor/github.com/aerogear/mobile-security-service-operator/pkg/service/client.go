package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aerogear/mobile-security-service/pkg/models"
	"github.com/go-logr/logr"
)

//DeleteAppFromServiceByRestAPI delete the app object in the service
//var function declaration to allow for local test mocking
var DeleteAppFromServiceByRestAPI = func(serviceAPI string, id string, reqLogger logr.Logger) error {
	reqLogger.Info("Calling REST Service to DELETE app", "serviceAPI", serviceAPI, "App.id", id)
	//Create the DELETE request
	url := serviceAPI + "/apps/" + id
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		reqLogger.Error(err, "Unable to create DELETE request", "HTTPMethod", http.MethodDelete, "url", url)
		return err
	}

	//Do the request
	client := &http.Client{}
	response, err := client.Do(req)

	if err != nil || response == nil || 204 != response.StatusCode {
		if response != nil {
			reqLogger.Error(err, "Unable to execute GET request", "HTTPMethod", http.MethodGet, "url", url, "response.StatusCode", response.StatusCode)
		}
		reqLogger.Error(err, "Unable to execute GET request", "HTTPMethod", http.MethodGet, "url", url)
		return err
	}

	defer response.Body.Close()

	reqLogger.Info("Successfully deleted app  ...", "App.Id:", id)
	return nil
}

//CreateAppByRestAPI create the app object in the service
//var function declaration to allow for local test mocking
var CreateAppByRestAPI = func(serviceAPI string, app *models.App, reqLogger logr.Logger) error {
	reqLogger.Info("Calling Service to POST app", "serviceAPI", serviceAPI, "App", app)

	// Create the object and parse for JSON
	appJSON, err := json.Marshal(app)
	if err != nil {
		reqLogger.Error(err, "Error to transform the app object in JSON", "App", app, "Error", err)
		return err
	}

	//Create the POST request
	url := serviceAPI + "/apps"
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(appJSON)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		reqLogger.Error(err, "Unable to create POST request", "HTTPMethod", http.MethodPost, "url", url)
		return err
	}

	//Do the request
	client := &http.Client{}
	response, err := client.Do(req)

	if err != nil || response == nil || 201 != response.StatusCode {
		if response != nil {
			reqLogger.Error(err, "Unable to execute GET request", "HTTPMethod", http.MethodGet, "url", url, "response.StatusCode", response.StatusCode)
		}
		reqLogger.Error(err, "Unable to execute GET request", "HTTPMethod", http.MethodGet, "url", url)
		return err
	}

	defer response.Body.Close()

	reqLogger.Info("Successfully created app  ...", "App:", app)
	return nil
}

//GetAppFromServiceByRestApi returns the app object from the service
func GetAppFromServiceByRestApi(serviceAPI string, appId string, reqLogger logr.Logger) (*models.App, error) {
	// Fill the record with the data from the JSON
	// Transform the body request in the version struct

	// Handle attempted request with no value passed for appID
	if appId == "" {
		err := fmt.Errorf("Cannot get App without AppId")
		return nil, err
	}

	//Create the GET request
	url := serviceAPI + "/apps" + "?appId=" + appId
	req, err := http.NewRequest(http.MethodGet, url, nil)
	reqLogger.Info("URL to get", "HTTPMethod", http.MethodGet, "url", url)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		reqLogger.Error(err, "Unable to create GET request", "HTTPMethod", http.MethodGet, "Request", req, "url", url)
		return nil, err
	}

	//Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		if response != nil {
			reqLogger.Error(err, "Unable to execute GET request", "HTTPMethod", http.MethodGet, "url", url, "response.StatusCode", response.StatusCode)
		}
		reqLogger.Error(err, "Unable to execute GET request", "HTTPMethod", http.MethodGet, "url", url)
		return nil, err
	}

	var obj []models.App
	err = json.NewDecoder(response.Body).Decode(&obj)

	app := models.App{}
	if err == io.ErrUnexpectedEOF {
		reqLogger.Error(err, "The app was not found in the REST Service API - Empty Response", "HTTPMethod", http.MethodGet, "url", url, "Response.Body", response.Body)
		return &app, nil
	}

	defer response.Body.Close()
	if 204 == response.StatusCode {
		reqLogger.Info("The app was not found in the REST Service API", "HTTPMethod", http.MethodGet, "url", url)
		return &app, nil
	}

	app = obj[0]
	reqLogger.Info("App found in the Service", "App", app)
	return &app, nil
}

//UpdateAppNameByRestAPI will update name of the APP in the Service
//var function declaration to allow for local test mocking
var UpdateAppNameByRestAPI = func(serviceAPI string, app *models.App, reqLogger logr.Logger) error {

	//Create the DELETE request
	url := serviceAPI + "/apps/" + app.ID
	appJSON, err := json.Marshal(app)

	if err != nil {
		reqLogger.Error(err, "Error to transform the app object in JSON", "AppJSON", appJSON, "App", app, "Error", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, url, strings.NewReader(string(appJSON)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		reqLogger.Error(err, "Unable to create PATCH request to update app name", "HTTPMethod", http.MethodPatch, "url", url)
		return err
	}
	reqLogger.Info("Calling Service to update app name", "serviceAPI", serviceAPI, "App", app)

	//Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil || response == nil || 204 != response.StatusCode {
		if response != nil {
			reqLogger.Error(err, "HTTP StatusCode not expected", "HTTPMethod", http.MethodPatch, "url", url, "response.StatusCode", response.StatusCode)
		}
		reqLogger.Error(err, "HTTP StatusCode not expected", "HTTPMethod", http.MethodPatch, "url", url)
		return err
	}
	defer response.Body.Close()

	reqLogger.Info("Successfully updated app name ...", "App", app)
	return nil
}
