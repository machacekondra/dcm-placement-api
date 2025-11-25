package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	apiserver "github.com/dcm-project/dcm-placement-api/internal/api_server"
	providerserver "github.com/dcm-project/dcm-placement-api/internal/api_server/providerserver"
	"github.com/dcm-project/dcm-placement-api/internal/config"
	"github.com/dcm-project/dcm-placement-api/internal/store"
	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	err := runCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the planner api",
	RunE: func(cmd *cobra.Command, args []string) error {
		defer zap.S().Info("API service stopped")

		cfg, err := config.New()
		if err != nil {
			zap.S().Fatalw("reading configuration", "error", err)
		}

		zap.S().Info("Starting API service...")
		zap.S().Info("Initializing data store")
		db, err := store.InitDB(cfg)
		if err != nil {
			zap.S().Fatalw("initializing data store", "error", err)
		}

		store := store.NewStore(db)
		defer store.Close()

		// SEED basic VM resource if not exists
		err = seedResources(store)
		if err != nil {
			zap.S().Fatalw("seeding resources", "error", err)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

		go func() {
			defer cancel()
			listener, err := newListener(cfg.Service.Address)
			if err != nil {
				zap.S().Fatalw("creating listener", "error", err)
			}

			server := apiserver.New(cfg, store, listener)
			if err := server.Run(ctx); err != nil {
				zap.S().Fatalw("Error running server", "error", err)
			}
		}()

		go func() {
			defer cancel()
			listener, err := newListener(cfg.Provider.Address)
			if err != nil {
				zap.S().Fatalw("creating listener", "error", err)
			}

			server := providerserver.New(cfg, store, listener)
			if err := server.Run(ctx); err != nil {
				zap.S().Fatalw("Error running provider server", "error", err)
			}
		}()

		<-ctx.Done()

		return nil
	},
}

func newListener(address string) (net.Listener, error) {
	if address == "" {
		address = "localhost:0"
	}
	return net.Listen("tcp", address)
}

func seedResources(store store.Store) error {
	definitionFiles := []string{
		"definitions/vm.yaml",
		"definitions/container.yaml",
	}
	for _, defFile := range definitionFiles {
		content, err := os.ReadFile(defFile)
		if err != nil {
			return err
		}

		// Unmarshal YAML into map[interface{}]interface{} first
		var yamlData map[interface{}]interface{}
		err = yaml.Unmarshal(content, &yamlData)
		if err != nil {
			return err
		}

		// Convert YAML structure to JSON-compatible format recursively
		converted := convertYAMLToJSON(yamlData)
		schema, ok := converted.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to convert YAML to map[string]interface{}")
		}

		resourceType := strings.TrimSuffix(filepath.Base(defFile), filepath.Ext(defFile))
		existing, getErr := store.Resource().Get(context.Background(), resourceType)
		if getErr == nil && existing != nil {
			_, createErr := store.Resource().Update(context.Background(), model.Resource{
				ID:           existing.ID,
				ResourceType: resourceType,
				Content:      schema,
			})
			if createErr != nil {
				return createErr
			}
		} else {
			_, createErr := store.Resource().Create(context.Background(), &model.Resource{
				ResourceType: resourceType,
				Content:      schema,
				ID:           uuid.New(),
			})
			if createErr != nil {
				return createErr
			}
		}
	}
	return nil
}

// convertYAMLToJSON recursively converts YAML types (map[interface{}]interface{}, []interface{})
// to JSON-compatible types (map[string]interface{}, []interface{} with converted elements)
func convertYAMLToJSON(data interface{}) interface{} {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			key := k.(string)
			result[key] = convertYAMLToJSON(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = convertYAMLToJSON(val)
		}
		return result
	default:
		return v
	}
}
