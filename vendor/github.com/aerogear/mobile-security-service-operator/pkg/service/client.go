package service

import (
	"encoding/json"
	"github.com/aerogear/mobile-security-service-operator/pkg/models"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	"github.com/go-logr/logr"
	"net/http"
	"strings"
)

//DeleteAppFromServiceByRestAPI delete the app object in the service
func DeleteAppFromServiceByRestAPI(protocol, host, hostSufix, id string, reqLogger logr.Logger ) error {
	reqLogger.Info( "Calling REST API to DELETE app", "protocol", protocol, "host", host, "hostSufix", hostSufix, "App.id", id )
	//Create the DELETE request
	url:= utils.GetRestAPIForApps(protocol, host, hostSufix)+"/"+id
	req, err := http.NewRequest(http.MethodDelete, url ,nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		reqLogger.Error(err, "Error when try to create the request", "HTTPMethod", http.MethodDelete, "Request", req, "url", url, "error", err)
		return err
	}

	//Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	defer response.Body.Close()

	if err != nil {
		reqLogger.Error(err, "Error when try to do the request", "HTTPMethod", http.MethodDelete, "Request", req, "url", url, "error", err)
		return err
	}

	reqLogger.Info("Deleted successfully app object in REST Service API",  "App.Id:", id)
	return nil
}


//CreateAppByRestAPI create the app object in the service
func CreateAppByRestAPI(protocol, host, hostSufix string, app models.App, reqLogger logr.Logger) error {
	reqLogger.Info( "Calling REST API to POST app", "protocol", protocol, "host", host, "hostSufix", hostSufix, "App", app )

	// Create the object and parse for JSON
	appJSON, err := json.Marshal(app)
	if err != nil {
		reqLogger.Error(err, "Error to transform the app object in JSON", "App", app, "Error", err)
		return err
	}

	//Create the POST request
	url:= utils.GetRestAPIForApps(protocol, host, hostSufix)
	req, err := http.NewRequest(http.MethodPost, url , strings.NewReader(string(appJSON)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		reqLogger.Error(err, "Error when try to create the request", "HTTPMethod", http.MethodPost, "Request", req, "url", url, "error", err)
		return err
	}

	//Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		reqLogger.Error(err, "Error when try to do the request", "HTTPMethod", http.MethodPost, "Request", req, "Response", response, "url", url, "error", err)
		return err
	}
	defer response.Body.Close()

	reqLogger.Info("Created successfully app object in REST Service API",  "App:", app)
	return nil
}

//GetAppFromServiceByRestApi returns the app object from the service
func GetAppFromServiceByRestApi(protocol, host, hostSufix, appId string, reqLogger logr.Logger) (models.App, error) {
	reqLogger.Info( "Calling REST API to GET app", "protocol", protocol, "host", host, "hostSufix", hostSufix, "App.appId", appId )

	// Fill the record with the data from the JSON
	// Transform the body request in the version struct
	got := models.App{}

	if len(appId) < 1 {
		reqLogger.Info( "Unable to get the AppID", "AppId", appId)
		return got, nil
	}

	//Create the GET request
	url:= utils.GetRestAPIForApps(protocol, host, hostSufix)+"?appId="+appId
	req, err := http.NewRequest(http.MethodGet,  url, nil)
	reqLogger.Info("URL to get", "HTTPMethod", http.MethodGet, "url", url)

	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		reqLogger.Error(err, "Error when try to create the request", "HTTPMethod", http.MethodGet, "Request", req, "url", url, "error", err)
		return got, err
	}

	//Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	defer response.Body.Close()

	if err != nil {
		reqLogger.Error(err, "Error when try to do the request", "HTTPMethod", http.MethodGet, "Request", req, "url", url, "error", err)
		return got, err
	}

	if 200 != response.StatusCode {
		reqLogger.Info("The app was not found in the REST Service API", "HTTPMethod", http.MethodGet, "url", url)
		return got, nil
	}

	var obj []models.App
	err = json.NewDecoder(response.Body).Decode(&obj)
	if err != nil {
		reqLogger.Error(err, "Error when try to do decoder the body response", "HTTPMethod", http.MethodGet, "Request", req, "Response", response, "url", url, "error", err)
		return got, err
	}

	if len(got.ID) < 0 {
		reqLogger.Info("The app was not found in the REST Service API", "HTTPMethod", http.MethodGet, "url", url)
		return got, nil
	}

	reqLogger.Info("App found in the Rest Service", "App", obj[0])
	return obj[0], nil
}

//deleteAppFromServiceByRestAPI delete the app object in the service
func UpdateAppNameByRestAPI(protocol, host, hostSufix string, app models.App, reqLogger logr.Logger) error {
	reqLogger.Info( "Calling REST API to PATCH app", "protocol", protocol, "host", host, "hostSufix", hostSufix, "App", app )

	//Create the DELETE request
	url:= utils.GetRestAPIForApps(protocol, host, hostSufix)+"/"+app.ID
	appJSON, err := json.Marshal(app)

	if err != nil {
		reqLogger.Error(err, "Error to transform the app object in JSON", "AppJSON", appJSON, "App", app, "Error", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, url ,strings.NewReader(string(appJSON)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		reqLogger.Error(err, "Error when try to create the request", "HTTPMethod", http.MethodPatch, "Request", req, "url", url, "error", err)
		return err
	}

	//Do the request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil || 204 != response.StatusCode {
		reqLogger.Error(err, "Error when try to do the request", "HTTPMethod", http.MethodPatch, "Request", req, "url", url, "error", err)
		return err
	}
	defer response.Body.Close()

	reqLogger.Info("Deleted successfully app object in REST Service API",  "App:", app)
	return nil
}