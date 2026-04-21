package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed wails.json
var wailsConfigData []byte

type WailsConfig struct {
	Name   string `json:"name"`
	Author struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Website string `json:"website"`
	} `json:"author"`
	Info struct {
		ProductName    string `json:"productName"`
		ProductVersion string `json:"productVersion"`
		Description    string `json:"description"`
		Copyright      string `json:"copyright"`
		Comments       string `json:"comments"`
	} `json:"info"`
}

var AppVersion WailsConfig

func init() {
	if err := json.Unmarshal(wailsConfigData, &AppVersion); err != nil {
		AppVersion.Info.ProductVersion = "unknown"
		AppVersion.Info.ProductName = "Proxy Checker"
	}
}

func GetVersionString() string {
	return AppVersion.Info.ProductVersion
}

func GetFullVersionInfo() map[string]interface{} {
	return map[string]interface{}{
		"version":     AppVersion.Info.ProductVersion,
		"productName": AppVersion.Info.ProductName,
		"description": AppVersion.Info.Description,
		"author":      AppVersion.Author.Name,
		"email":       AppVersion.Author.Email,
		"website":     AppVersion.Author.Website,
		"copyright":   AppVersion.Info.Copyright,
		"comments":    AppVersion.Info.Comments,
	}
}
