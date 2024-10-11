package playgrounds

import (
	"atlas-cli-plugin/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const BASE_URL = "https://search-playground.mongodb.com/api/tools/code-playground/snapshots"

type Playground struct {
	SnapshotID   string `json:"snapshotId"`
	Name         string `json:"name"`
	SearchConfig struct {
		AggregationPipeline string `json:"aggregationPipeline"`
		IndexDefinition     string `json:"indexDefinition"`
		Documents           string `json:"documents"`
		Synonyms            string `json:"synonyms"`
	} `json:"searchConfig"`
	RetainIndefinitely bool `json:"retainIndefinitely"`
}

func fetchPlayground(snapshotID string) (*Playground, error) {
	url := fmt.Sprintf("%s/%s", BASE_URL, snapshotID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch playground: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var playground Playground
	if err := json.Unmarshal(body, &playground); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &playground, nil
}

func storeDataIntoMongoDB(connectionString string, databaseName string, collectionName string, data string) error {

	var jsonArray []interface{}
	if err := json.Unmarshal([]byte(data), &jsonArray); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	// Convert the JSON array to a BSON array
	bsonArray := bson.A(jsonArray)

	// Connect to MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionString))
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer client.Disconnect(ctx)

	// Insert the BSON array into MongoDB
	collection := client.Database(databaseName).Collection(collectionName)
	_, err = collection.InsertMany(ctx, bsonArray)

	if err != nil {
		return fmt.Errorf("failed to insert data into MongoDB: %w", err)
	}

	return nil
}

func toCamelCase(s string) string {
	// Split the string into words
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	// Convert the first word to lowercase
	if len(words) == 0 {
		return ""
	}
	words[0] = strings.ToLower(words[0])

	// Capitalize the first letter of each subsequent word
	for i := 1; i < len(words); i++ {
		words[i] = strings.Title(words[i])
	}

	// Join the words back together
	return strings.Join(words, "")
}

func ImportCmdBuilder() *cobra.Command {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a playground",
		RunE: func(cmd *cobra.Command, args []string) error {
			urlOrSnapshotID := cmd.Flags().Arg(0)
			deploymentName, _ := cmd.Flags().GetString("deploymentName")
			dbuser, _ := cmd.Flags().GetString("dbuser")
			dbpass, _ := cmd.Flags().GetString("dbpass")
			debug, _ := cmd.Flags().GetBool("debug")

			if !debug {
				log.SetOutput(io.Discard)
			}

			atlasCliExe := utils.AtlasCliExe()
			atlasCmd := exec.Command(atlasCliExe, "deployments", "connect", deploymentName, "--connectWith", "connectionString")
			atlasCmd.Env = os.Environ()
			var stdout bytes.Buffer
			atlasCmd.Stdout = &stdout

			if err := atlasCmd.Run(); err != nil {
				log.Fatalf("Error running command: %v", err)
			}

			connectionString := stdout.String()
			connectionString = connectionString[:len(connectionString)-1]

			if dbuser != "" || dbpass != "" {
				uri, err := url.Parse(connectionString)
				if err != nil {
					log.Fatalf("Error parsing connection string: %v", err)
				}

				uri.User = url.UserPassword(dbuser, dbpass)
				connectionString = uri.String()
			}

			log.Printf("Connection String: %s\n", connectionString)

			// if the user provided a URL, extract the snapshot ID
			// the URL should be in the format: https://search-playground.mongodb.com/api/tools/code-playground/snapshots/<snapshot_id>
			parsedURL, err := url.Parse(urlOrSnapshotID)
			if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
				// Not a URL, assume it's a snapshot ID
				snapshotID := urlOrSnapshotID
				playground, err := fetchPlayground(snapshotID)
				if err != nil {
					return err
				}
				fmt.Printf("Fetched playground: %+v\n", playground)
			} else {
				// It's a URL, extract the snapshot ID
				pathSegments := strings.Split(parsedURL.Path, "/")
				snapshotID := pathSegments[len(pathSegments)-1]
				playground, err := fetchPlayground(snapshotID)
				if err != nil {
					return err
				}
				fmt.Printf("Fetched playground: %+v\n", playground)
				if err := storeDataIntoMongoDB(connectionString, "playground", toCamelCase(playground.Name), playground.SearchConfig.Documents); err != nil {
					return err
				}
			}

			return nil
		},
	}

	importCmd.Flags().String("deploymentName", "", "Name of the deployment where the playground data will be imported")
	importCmd.MarkFlagRequired("deploymentName")
	importCmd.Flags().String("dbuser", "", "Database user")
	importCmd.Flags().String("dbpass", "", "Database password")
	importCmd.Flags().Bool("debug", false, "Enable debug mode")

	return importCmd
}
