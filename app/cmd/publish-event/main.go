package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/segmentio/kafka-go"
	"gopkg.in/yaml.v3"
)

type appConfig struct {
	Trigger struct {
		Kafka struct {
			Brokers []string `yaml:"brokers"`
			Topic   string   `yaml:"topic"`
		} `yaml:"kafka"`
		SQS struct {
			QueueURL string `yaml:"queue_url"`
			Endpoint string `yaml:"endpoint"`
			Region   string `yaml:"region"`
		} `yaml:"sqs"`
	} `yaml:"trigger"`
}

func main() {
	configPath := flag.String("config", "../config.yaml", "path to config.yaml")
	payloadPath := flag.String("payload", "../examples/payment-event.json", "path to event payload JSON")
	target := flag.String("target", "both", "kafka, sqs or both")
	count := flag.Int("count", 1, "number of events to publish")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	exitOnError(err)

	payload, err := os.ReadFile(*payloadPath)
	exitOnError(err)
	payload = bytes.TrimSpace(payload)
	exitOnError(validateJSON(payload))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < *count; i++ {
		switch strings.ToLower(strings.TrimSpace(*target)) {
		case "kafka":
			exitOnError(publishKafka(ctx, cfg, payload))
		case "sqs":
			exitOnError(publishSQS(ctx, cfg, payload))
		case "both":
			exitOnError(publishKafka(ctx, cfg, payload))
			exitOnError(publishSQS(ctx, cfg, payload))
		default:
			exitOnError(fmt.Errorf("invalid target %q: use kafka, sqs or both", *target))
		}
	}
}

func loadConfig(path string) (appConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return appConfig{}, err
	}
	data = []byte(expandEnvDefaults(string(data)))

	var cfg appConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return appConfig{}, err
	}
	if len(cfg.Trigger.Kafka.Brokers) == 0 {
		cfg.Trigger.Kafka.Brokers = []string{"localhost:9092"}
	}
	if len(cfg.Trigger.Kafka.Brokers) == 1 && strings.Contains(cfg.Trigger.Kafka.Brokers[0], ",") {
		cfg.Trigger.Kafka.Brokers = splitCSV(cfg.Trigger.Kafka.Brokers[0])
	}
	if cfg.Trigger.Kafka.Topic == "" {
		cfg.Trigger.Kafka.Topic = "payment-events"
	}
	if cfg.Trigger.SQS.Region == "" {
		cfg.Trigger.SQS.Region = "us-east-1"
	}
	if cfg.Trigger.SQS.Endpoint == "" {
		cfg.Trigger.SQS.Endpoint = "http://localhost:4566"
	}
	if cfg.Trigger.SQS.QueueURL == "" {
		cfg.Trigger.SQS.QueueURL = "http://localhost:4566/000000000000/payment-events"
	}
	return cfg, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func expandEnvDefaults(input string) string {
	var out strings.Builder
	for i := 0; i < len(input); i++ {
		if input[i] != '$' || i+1 >= len(input) || input[i+1] != '{' {
			out.WriteByte(input[i])
			continue
		}

		end := strings.IndexByte(input[i+2:], '}')
		if end < 0 {
			out.WriteByte(input[i])
			continue
		}

		expr := input[i+2 : i+2+end]
		out.WriteString(expandEnvExpression(expr))
		i += end + 2
	}
	return out.String()
}

func expandEnvExpression(key string) string {
	if name, fallback, ok := strings.Cut(key, ":-"); ok {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
		return fallback
	}
	return os.Getenv(key)
}

func validateJSON(payload []byte) error {
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return fmt.Errorf("invalid payload JSON: %w", err)
	}
	return nil
}

func publishKafka(ctx context.Context, cfg appConfig, payload []byte) error {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Trigger.Kafka.Brokers...),
		Topic:        cfg.Trigger.Kafka.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}
	defer writer.Close()

	err := writer.WriteMessages(ctx, kafka.Message{
		Value: payload,
		Headers: []kafka.Header{
			{Key: "source", Value: []byte("publish-event")},
			{Key: "content-type", Value: []byte("application/json")},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("published kafka event topic=%s brokers=%s\n", cfg.Trigger.Kafka.Topic, strings.Join(cfg.Trigger.Kafka.Brokers, ","))
	return nil
}

func publishSQS(ctx context.Context, cfg appConfig, payload []byte) error {
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Trigger.SQS.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")),
		awsconfig.WithBaseEndpoint(cfg.Trigger.SQS.Endpoint),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return err
	}

	body := string(payload)
	client := sqs.NewFromConfig(awsCfg)
	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &cfg.Trigger.SQS.QueueURL,
		MessageBody: &body,
	})
	if err != nil {
		return err
	}
	fmt.Printf("published sqs event queue=%s endpoint=%s\n", cfg.Trigger.SQS.QueueURL, cfg.Trigger.SQS.Endpoint)
	return nil
}

func exitOnError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
