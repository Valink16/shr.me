package main

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"
)

type HtmlTemplate struct {
	tmplt *template.Template
}

// Loads the file and adds it's content to the single field in the template and returns the output
func (htmlTmplt *HtmlTemplate) ApplyToHtmlFile(pathname string) ([]byte, error) {
	data, err := os.ReadFile(pathname)
	if err != nil {
		return []byte{}, err
	}

	var output bytes.Buffer
	htmlTmplt.tmplt.Execute(&output, template.HTML(data))
	return output.Bytes(), nil
}

// Adds the provided data to the single field in the template and returns the output
func (htmlTmplt *HtmlTemplate) ApplyToHtml(data []byte) ([]byte, error) {
	var output bytes.Buffer
	htmlTmplt.tmplt.Execute(&output, template.HTML(data))
	return output.Bytes(), nil
}

// Adds the provided data to the single field in the template and returns the output
func (htmlTmplt *HtmlTemplate) ApplyToData(data any) ([]byte, error) {
	var output bytes.Buffer
	htmlTmplt.tmplt.Execute(&output, data)
	return output.Bytes(), nil
}

// Writes the http response with the template applied to a file
func (htmlTmplt *HtmlTemplate) WriteFile(pathname string, w http.ResponseWriter) {
	data, err := htmlTmplt.ApplyToHtmlFile(pathname)
	if err != nil {
		Error.Println("Failed to apply template", err)
		w.WriteHeader(404)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

// Writes the http response with the template applied to a file
func (htmlTmplt *HtmlTemplate) WriteData(data any, w http.ResponseWriter) {
	newData, err := htmlTmplt.ApplyToData(data)

	if err != nil {
		Error.Println("Failed to apply template", err)
		w.WriteHeader(404)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(newData)
}

func loadTemplateFile(pathname string) (*HtmlTemplate, error) {
	name := strings.Split(path.Base(pathname), ".")[0]
	data, err := os.ReadFile(pathname)
	if err != nil {
		return nil, err
	}

	tmplt, err := template.New(name).Parse(string(data))
	if err != nil {
		return nil, err
	}

	return &HtmlTemplate{tmplt: tmplt}, nil
}

func loadTemplate(name, templateStr string) (*HtmlTemplate, error) {
	tmplt, err := template.New(name).Parse(templateStr)
	if err != nil {
		return nil, err
	}

	return &HtmlTemplate{tmplt: tmplt}, nil
}
