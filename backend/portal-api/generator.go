package main

import assets "internal-developer-portal/portal-api/templates"

func readGoTemplate(fileName string) (string, error) { return assets.Read("go-gin-api", fileName) }
func renderGoTemplate(fileName, moduleName string) (string, error) {
	return assets.Render("go-gin-api", fileName, map[string]string{"ModuleName": moduleName})
}
