package main

import (
	"allora-network/forecast-data-provider/types"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Command struct {
	Parts []string
}

type ClientConfig struct {
	Node     string
	CliApp   string
	Commands map[string]Command
}

var config ClientConfig
var workersNum uint
var awsAccessKey string
var awsSecretKey string
var s3BucketName string
var s3FileKey string
var resetDB bool
var parallelJobs uint

func ExecuteCommand(cliApp, node string, parts []string) ([]byte, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("no command parts provided")
	}

	var completeParts []string
	for _, part := range parts {
		completeParts = append(completeParts, part)
	}

	completeParts = replacePlaceholders(completeParts, "{node}", node)
	completeParts = replacePlaceholders(completeParts, "{cliApp}", cliApp)

	log.Debug().Strs("command", completeParts).Msg("Executing command")
	cmd := exec.Command(completeParts[0], completeParts[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(output)).Msg("Command execution failed")
		return nil, err
	}

	return output, nil
}

func replacePlaceholders(parts []string, placeholder, value string) []string {
	var replacedParts []string
	for _, part := range parts {
		if part == placeholder {
			replacedParts = append(replacedParts, value)
		} else {
			replacedParts = append(replacedParts, part)
		}
	}
	return replacedParts
}

func ExecuteCommandByKey[T any](config ClientConfig, key string, params ...string) (T, error) {
	var result T

	cmd, ok := config.Commands[key]
	if !ok {
		return result, fmt.Errorf("command not found")
	}

	if len(params) > 0 {
		cmd.Parts = append(cmd.Parts, params...)
	}

	log.Debug().Str("commandName", key).Msg("Starting execution")
	output, err := ExecuteCommand(config.CliApp, config.Node, cmd.Parts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute command")
		return result, err
	}

	log.Debug().Str("raw output", string(output)).Msg("Command raw output")

	err = json.Unmarshal(output, &result)
	if err != nil {
		log.Error().Err(err).Str("json", string(output)).Msg("Failed to unmarshal JSON")
		return result, err
	}

	return result, nil
}

func DecodeTx(config ClientConfig, params string) (types.Tx, error) {
	var result types.Tx

	result, err := ExecuteCommandByKey[types.Tx](config, "decodeTx", params)
	if err == nil {
		return result, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return types.Tx{}, err
	}
	root := filepath.Join(dir, "previous")
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "allorad" {
			fmt.Println("Trying decode with file:", path)
			config.CliApp = path
			decodeTx, err := ExecuteCommandByKey[types.Tx](config, "decodeTx", params)
			if err == nil {
				result = decodeTx
			}
		}
		return nil
	})

	return result, err
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		nodeFlag       string
		cliAppFlag     string
		connectionFlag string
	)

	pflag.UintVar(&workersNum, "WORKERS_NUM", 5, "Number of workers to process blocks concurrently")
	pflag.StringVar(&nodeFlag, "NODE", "https://allora-rpc.devnet.behindthecurtain.xyz", "Node address") //# https://default-node-address:443",
	pflag.StringVar(&cliAppFlag, "CLIAPP", "allorad", "CLI app to execute commands")
	pflag.StringVar(&connectionFlag, "CONNECTION", "postgres://app:app@localhost:5433/app", "Database connection string")
	pflag.StringVar(&awsAccessKey, "AWS_ACCESS_KEY", "", "AWS access key")
	pflag.StringVar(&awsSecretKey, "AWS_SECURITY_KEY", "", "AWS security key")
	pflag.StringVar(&s3BucketName, "S3_BUCKET_NAME", "allora-testnet-1-indexer-backups", "AWS s3 bucket name")
	pflag.StringVar(&s3FileKey, "S3_FILE_KEY", "pgdump-20240723-21-53-11.dump", "AWS s3 file key")
	pflag.BoolVar(&resetDB, "RESET_DB", false, "Database reset flag")
	pflag.UintVar(&parallelJobs, "RESTORE_PARALLEL_JOBS", 4, "Database reset flag")
	pflag.Parse()

	log.Info().
		Uint("WORKERS_NUM", workersNum).
		Str("NODE", nodeFlag).
		Str("CONNECTION", connectionFlag).
		Str("AWS_ACCESS_KEY", awsAccessKey).
		Str("AWS_SECURITY_KEY", awsSecretKey).
		Str("S3_BUCKET_NAME", s3BucketName).
		Str("S3_BUCKET_NAME", s3FileKey).
		Msg("provider started")

	// define the commands to execute payloads
	config = ClientConfig{
		Node:   nodeFlag,
		CliApp: cliAppFlag,
		Commands: map[string]Command{
			"latestBlock": {
				Parts: []string{"{cliApp}", "query", "consensus", "comet", "block-latest", "--node", "{node}", "--output", "json"},
			},
			"blockByHeight": { // Add a template command for fetching blocks by height
				Parts: []string{"{cliApp}", "query", "block", "--type=height", "--node", "{node}", "--output", "json", "{height}"},
			},
			"consensusParams": {
				Parts: []string{"{cliApp}", "query", "consensus", "params", "--node", "{node}", "--output", "json"},
			},
			"decodeTx": {
				Parts: []string{"{cliApp}", "tx", "decode", "--node", "{node}", "--output", "json"}, // Requires , "{txData}"
			},
			"nextTopicId": {
				Parts: []string{"{cliApp}", "query", "emissions", "next-topic-id", "--node", "{node}", "--output", "json"},
			},
			"topicById": {
				Parts: []string{"{cliApp}", "query", "emissions", "topic", "--node", "{node}", "--output", "json"}, // Requires "{topic}"
			},
		},
	}

	// Init DB
	initDB(connectionFlag, resetDB)
	defer closeDB()

	_, err := downloadBackupFromS3()
	if err != nil {
		log.Log().Err(err).Msg("Failed restoring DB and start fetching blockchain data from scratch")
		setupDB()
	}
	// Set a cancel context to stop the workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up a channel to listen for block heights to process
	heightsChan := make(chan uint64, workersNum)
	defer close(heightsChan)

	wgBlocks := sync.WaitGroup{}
	for j := uint(1); j <= workersNum; j++ {
		wgBlocks.Add(1)
		go worker(ctx, &wgBlocks, heightsChan)
	}
	defer wgBlocks.Wait() // Wait for all workers to finish at the end of the main function

	// If no blocks are provided, start the main loop
	log.Info().Msg("Starting main loop...")
	generateBlocksLoop(ctx, heightsChan)

	log.Info().Msg("Exited main loop, waiting for subroutines to finish...")
	cancel()
}

// Generates the block heights to process in an infinite loop
func generateBlocksLoop(ctx context.Context, heightsChan chan<- uint64) {
	for {
		lastProcessedHeight, err := getLatestBlockHeightFromDB()
		if err != nil {
			log.Error().Err(err).Msg("Failed to getLatestBlockHeightFromDB")
		}
		chainLatestHeight, err := getLatestHeight()
		if err != nil {
			log.Error().Err(err).Msg("Failed to getLatestHeight from chain")
		}
		log.Info().Msgf("Processing heights from %d to %d", lastProcessedHeight, chainLatestHeight)
		// Emit heights to process into channel
		for w := lastProcessedHeight; w <= chainLatestHeight; w++ {
			select {
			case <-ctx.Done():
				log.Info().Msg("Context cancelled, exiting...")
				return
			default:
				heightsChan <- w
			}
		}
		log.Info().Msg("All blocks processed...")
		time.Sleep(10 * time.Second)
	}
}

func worker(ctx context.Context, wgBlocks *sync.WaitGroup, heightsChan <-chan uint64) {
	defer wgBlocks.Done()
	for {
		select {
		case <-ctx.Done():
			// Context was cancelled, stop the worker
			return
		case height, ok := <-heightsChan:
			if !ok {
				// heightsChan was closed, stop the worker
				log.Warn().Msg("heightsChan closed, stopping worker")
				return
			}
			// Tx
			log.Info().Msgf("Processing height: %d", height)
			block, err := fetchBlock(config, height)
			if err != nil {
				log.Error().Err(err).Msg("Worker: Failed to fetchBlock block height")
			}
			log.Info().Msgf("fetchBlock height: %d, len(TXs): %d", height, len(block.Data.Txs))
			err = writeBlock(config, block)
			if err != nil {
				log.Error().Err(err).Msgf("Worker: Failed to writeBlock, height: %d", height)
			}

			log.Info().Msgf("Write height: %d", height)

			if len(block.Data.Txs) > 0 {
				log.Info().Msgf("Processing txs at height: %d", height)
				wgTxs := sync.WaitGroup{}
				for _, encTx := range block.Data.Txs {
					wgTxs.Add(1)
					log.Info().Msgf("Processing height: %d, encTx: %s", height, encTx)
					go processTx(&wgTxs, height, encTx)
				}
				wgTxs.Wait()
			}

			// Events
			log.Info().Msgf("Processing height: %d", height)
			err = processBlock(config, height)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get block events")
			}
		}
	}

}
