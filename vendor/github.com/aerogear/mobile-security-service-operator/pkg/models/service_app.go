package models

type App struct{
	ID                    string     `json:"id"`
	AppID                 string     `json:"appId"`
	AppName               string     `json:"appName,omitempty"`
	DeletedAt             string     `json:"deletedAt,omitempty"`
}

func NewApp(name, id string) App {
	app := new(App)
	app.AppName = name
	app.AppID =id
	return *app
}