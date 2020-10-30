package main

import (
	cfg "config"
	"context"
	"lambdaapi"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"workers"

	"github.com/sirupsen/logrus"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = lambdaapi.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"), extensionName)
	logger          = logrus.New().WithField("Name", extensionName)
)
var producer workers.TaskProducer
var consumer workers.TaskConsumer
var config *cfg.LambdaExtensionConfig
var dataQueue chan []byte

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			consumer.FlushDataQueue()
			return
		default:
			consumer.DrainQueue(ctx)
			time.Sleep(5 * time.Second)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		logger.Info("Received", s)
	}()
	logger.Logger.SetOutput(os.Stdout)

	// Creating config and performing validation

	os.Setenv("MAX_RETRY", "3")
	os.Setenv("SUMO_HTTP_ENDPOINT", "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw==")
	os.Setenv("S3_BUCKET_NAME", "test-angad")
	os.Setenv("S3_BUCKET_REGION", "test-angad")
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "himlambda")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "Latest$")
	os.Setenv("ENABLE_FAILOVER", "true")
	os.Setenv("LOG_LEVEL", "5")
	os.Setenv("MAX_DATAQUEUE_LENGTH", "10")
	os.Setenv("MAX_CONCURRENT_REQUESTS", "2")

	config, _ = cfg.GetConfig()

	logger.Logger.SetLevel(config.LogLevel)
	logger.Debug(config)
	dataQueue = make(chan []byte, config.MaxDataQueueLength)

	// Start HTTP Server before subscription in a goRoutine
	// producer = workers.NewTaskProducer(dataQueue, logger)
	// go producer.Start()

	// Creating SumoTaskConsumer
	consumer = workers.NewTaskConsumer(dataQueue, config, logger)

	go func() {
		numDataGenerated := 5
		largedata := []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
		for i := 0; i < numDataGenerated; i++ {
			logger.Debugf("Producing data into dataQueue: %d", i+1)
			dataQueue <- largedata
			sleepTime := i % 4
			time.Sleep(time.Duration(sleepTime) * time.Second)
		}
		close(dataQueue)
		return
	}()
	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)
}
