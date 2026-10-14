package templates

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed go-gin-api/*.tmpl react-frontend/*.tmpl
var Files embed.FS

func Read(templateID, fileName string) (string, error) {
	data, err := Files.ReadFile(templateID + "/" + fileName)
	return string(data), err
}

func Render(templateID, fileName string, values any) (string, error) {
	source, err := Read(templateID, fileName)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New(fileName).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", err
	}
	var result bytes.Buffer
	err = tmpl.Execute(&result, values)
	return result.String(), err
}
