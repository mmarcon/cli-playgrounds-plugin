package utils

import (
	"embed"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/debme"
	"github.com/leaanthony/gosod"
)

//go:embed project_template/*
var projectTemplate embed.FS

type config struct {
	IndexName           string
	Mappings            string
	DatabaseName        string
	CollectionName      string
	AggregationPipeline string
}

func GenerateProject(projectDir string, mappings string, databaseName string, collectionName string, aggregationPipeline string) {
	root, _ := debme.FS(projectTemplate, "project_template")
	project := gosod.New(root)
	projectConfig := &config{
		IndexName:           "default",
		Mappings:            mappings,
		DatabaseName:        databaseName,
		CollectionName:      collectionName,
		AggregationPipeline: aggregationPipeline,
	}

	err := project.Extract(projectDir, projectConfig)

	if err != nil {
		log.Fatal(err)
	}

	err = walkAndFormat(projectDir)

	if err != nil {
		log.Fatal(err)
	}
}

func reformatFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var formattedContent string
	if strings.HasSuffix(path, ".json") {
		var obj map[string]interface{}
		json.Unmarshal(content, &obj)
		formattedBytes, _ := json.MarshalIndent(obj, "", "  ")
		formattedContent = string(formattedBytes)
	}

	//TODO: Add support for files other than JSON

	return os.WriteFile(path, []byte(formattedContent), 0644)
}

func walkAndFormat(rootDirectory string) error {
	return filepath.Walk(rootDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".json")) {
			log.Printf("Formatting file: %s\n", path)
			if err := reformatFile(path); err != nil {
				return err
			}
		}
		return nil
	})
}
