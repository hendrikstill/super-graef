package main

import (
	"os"
	"time"

	"github.com/amimof/huego"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
}

type Config struct {
	sqsQueueUrl           *string
	hueBridgeHost         *string
	hueBridgeUser         *string
	hueBridgeSwitchNumber *int
	millTime              *time.Duration
}

func main() {
	config := parseConfig()
	chnMessages := config.pollSqs()

	for message := range chnMessages {
		log.Info(message)
		config.runMill()
	}
}

func parseConfig() *Config {
	config := &Config{
		sqsQueueUrl:           kingpin.Flag("sqsQueueUrl", "SQS Queue URL").Envar("SQS_QUEUE_URL").Required().String(),
		hueBridgeHost:         kingpin.Flag("hueBridgeHost", "HueBridge address").Envar("HUE_BRIDGE_HOST").Required().String(),
		hueBridgeUser:         kingpin.Flag("hueBridgeUser", "HueBridge user").Envar("HUE_BRIDGE_USER").Required().String(),
		hueBridgeSwitchNumber: kingpin.Flag("hueBridgeSwitchNumber", "Hue Index of switch").Envar("HUE_BRIDGE_SWITCH_NUMBER").Default("12").Int(),
		millTime:              kingpin.Flag("millTime", "Mill duration").Envar("MILL_TIME").Default("9s").Duration(),
	}

	kingpin.Parse()

	log.Infof("%+v\n", config)
	return config
}

func (config *Config) pollSqs() <-chan *sqs.Message {
	chnMessages := make(chan *sqs.Message)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	sqsClient := sqs.New(sess)

	log.Infof("Start listening on SQS queue %s", *config.sqsQueueUrl)

	go func() {
		for {
			output, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            config.sqsQueueUrl,
				MaxNumberOfMessages: aws.Int64(1),
				WaitTimeSeconds:     aws.Int64(15),
			})

			if err != nil {
				log.Errorf("failed to fetch sqs message %v", err)
			}

			for _, message := range output.Messages {
				sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      config.sqsQueueUrl,
					ReceiptHandle: message.ReceiptHandle,
				})
				if err != nil {
					log.Errorf("failed to delete sqs message %v", err)
				}
				chnMessages <- message
			}
		}
	}()

	return chnMessages

}

func (config *Config) runMill() {
	bridge := huego.New(*config.hueBridgeHost, *config.hueBridgeUser)
	powerSwitch, _ := bridge.GetLight(*config.hueBridgeSwitchNumber)

	log.Infof("Switch on %d for %s", *config.hueBridgeSwitchNumber, config.millTime.String())

	powerSwitch.On()
	time.Sleep(*config.millTime)
	powerSwitch.Off()
	log.Info("Switch off")
}
