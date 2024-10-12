package utils

import (
	"bytes"
	"os"
	"text/template"
)

const searchIndexTemplate = `{
  "name": "{{.IndexName}}",
  "database": "{{.DatabaseName}}",
  "collectionName": "{{.CollectionName}}",
  "definition": {{.IndexDefinition}}
}`

const scriptTemplate = `
use("{{.DatabaseName}}");
db.{{.CollectionName}}.aggregate({{.AggregationPipeline}}).toArray();
`

// create a brand new directory for the project
func CreateProjectDir(projectDir string) error {
	return os.Mkdir(projectDir, 0755)
}

func StoreSearchIndexDefinition(projectDir string, indexDefinition string, databaseName string, collectionName string) error {
	// Set default value for indexName if not provided
	idxName := "default"

	tmpl, err := template.New("searchIndex").Parse(searchIndexTemplate)
	if err != nil {
		return err
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, map[string]string{
		"IndexName":       idxName,
		"DatabaseName":    databaseName,
		"CollectionName":  collectionName,
		"IndexDefinition": indexDefinition,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(projectDir+"/search-index.json", result.Bytes(), 0644)
}

func StoreScript(projectDir string, aggregationPipeline string, databaseName string, collectionName string) error {
	tmpl, err := template.New("script").Parse(scriptTemplate)
	if err != nil {
		return err
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, map[string]string{
		"DatabaseName":        databaseName,
		"CollectionName":      collectionName,
		"AggregationPipeline": aggregationPipeline,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(projectDir+"/playground.mongodb.js", result.Bytes(), 0644)
}
