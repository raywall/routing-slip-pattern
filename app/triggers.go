package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/raywall/routing-slip-pattern/slip"
	"github.com/segmentio/kafka-go"
)

const defaultShutdownTimeout = 5 * time.Second

var correlationFallbackCounter atomic.Uint64

type AppRuntime struct {
	config   *AppConfig
	workflow *WorkflowConfig
	logger   *slog.Logger
	router   *slip.Router
	store    slip.StateStore
	steps    []slip.StepDef
}

func newAppRuntime(cfg *AppConfig, workflow *WorkflowConfig, logger *slog.Logger) (*AppRuntime, error) {
	cfg.ApplyIntegrationEnv()

	policy, err := workflow.RoutingErrorPolicy()
	if err != nil {
		return nil, err
	}

	store, err := buildStateStore(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	router := buildRouterWithOptions(logger, policy,
		slip.WithStateStore(store),
		slip.WithStateOptions(stateOptions(cfg, workflow)),
		slip.WithMiddleware(slip.MetricsMiddleware(
			slip.HTTPMetricsEmitter{Endpoint: cfg.Metrics.Endpoint},
			slip.MetricsOptions{
				Workflow: workflow.Name,
				Segment:  cfg.Metrics.Segment,
				Source:   cfg.Metrics.Source,
				Tags:     cfg.Metrics.Tags,
			},
			logger,
		)),
	)
	router.MustRegister(&FlakyHandler{})

	return &AppRuntime{
		config:   cfg,
		workflow: workflow,
		logger:   logger,
		router:   router,
		store:    store,
		steps:    workflow.ToSlip(),
	}, nil
}

func runConfiguredApp(ctx context.Context, cfg *AppConfig, workflow *WorkflowConfig, logger *slog.Logger) error {
	runtime, err := newAppRuntime(cfg, workflow, logger)
	if err != nil {
		return err
	}

	logger.Info("configured workflow ready",
		slog.String("connector", cfg.Trigger.Connector),
		slog.String("mode", cfg.Trigger.Mode),
		slog.String("workflow", workflow.Name),
		slog.Int("steps", len(runtime.steps)),
	)

	if cfg.MCP.Enabled {
		go func() {
			if err := newMCPServer(cfg, workflow, runtime.store, logger).run(ctx); err != nil {
				logger.Error("mcp gateway stopped", slog.String("error", err.Error()))
			}
		}()
	}

	switch cfg.Trigger.Connector {
	case "rest":
		return runtime.runREST(ctx)
	case "kafka":
		return runtime.runKafka(ctx)
	case "sqs":
		return runtime.runSQS(ctx)
	case "sns":
		return runtime.runSNS(ctx)
	default:
		return fmt.Errorf("unsupported trigger connector %q", cfg.Trigger.Connector)
	}
}

func (r *AppRuntime) processPayload(ctx context.Context, payload map[string]any, headers map[string]string) (*slip.Message, error) {
	correlationID, hasCorrelation := stringFromPath(payload, r.workflow.CorrelationIDPath)
	if !hasCorrelation {
		correlationID = newCorrelationUUID()
		setPayloadPath(payload, r.workflow.CorrelationIDPath, correlationID)
	}

	messageID, ok := stringFromPath(payload, r.workflow.MessageIDPath)
	if !ok {
		messageID = correlationID
	}

	lease, acquired, err := r.acquireProcessingLease(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, fmt.Errorf("%w: %s", slip.ErrProcessingLocked, messageID)
	}
	if lease != nil {
		defer func() {
			if err := lease.Release(context.Background()); err != nil {
				r.logger.Warn("processing lease release failed",
					slog.String("message_id", messageID),
					slog.String("error", err.Error()),
				)
			}
		}()
	}

	msg := slip.NewMessage(messageID, payload)
	if snapshot, err := r.store.Load(ctx, messageID); err == nil {
		msg = slip.MessageFromSnapshot(snapshot)
	} else if !slip.IsStateNotFound(err) {
		return nil, fmt.Errorf("load state for message %q: %w", messageID, err)
	}

	for key, value := range headers {
		msg.Headers[key] = value
	}
	msg.Headers["workflow"] = r.workflow.Name
	if r.workflow.Version != "" {
		msg.Headers["workflow_version"] = r.workflow.Version
	}
	if msg.RemainingSteps() == 0 && msg.Cursor() == 0 {
		msg.AttachSlip(r.steps)
	}

	if msg.CorrelationID == "" {
		msg.CorrelationID = correlationID
		msg.Headers["correlation_id"] = correlationID
	} else {
		msg.Headers["correlation_id"] = msg.CorrelationID
	}
	if msg.Status == "completed" && msg.RemainingSteps() == 0 {
		r.logger.Info("message already completed; returning persisted snapshot",
			slog.String("message_id", msg.ID),
			slog.String("correlation_id", msg.CorrelationID),
		)
		return msg, nil
	}
	if trace, ok := slip.ParseTraceparent(msg.Headers["traceparent"]); ok {
		msg.TraceID = trace.TraceID
		msg.SpanID = trace.SpanID
	} else if traceID := firstHeader(msg.Headers, "trace_id", "x-trace-id", "X-Trace-ID"); traceID != "" {
		msg.TraceID = traceID
	}

	err = r.router.Process(ctx, msg)
	return msg, err
}

func (r *AppRuntime) acquireProcessingLease(ctx context.Context, messageID string) (slip.ProcessingLease, bool, error) {
	if r.config.StateStore.ProcessingLock.Enabled != nil && !*r.config.StateStore.ProcessingLock.Enabled {
		return nil, true, nil
	}
	locker, ok := r.store.(slip.ProcessingLocker)
	if !ok || locker == nil {
		return nil, true, nil
	}
	ttl := time.Duration(r.config.StateStore.ProcessingLock.TTLSeconds) * time.Second
	owner := fmt.Sprintf("%s:%s:%d", r.config.Service.Name, r.config.Service.RunID, os.Getpid())
	return locker.TryAcquireProcessing(ctx, messageID, owner, ttl)
}

func newCorrelationUUID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		seed := fmt.Sprintf("%d:%d:%d", time.Now().UnixNano(), os.Getpid(), correlationFallbackCounter.Add(1))
		sum := sha256.Sum256([]byte(seed))
		copy(bytes[:], sum[:16])
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}

func setPayloadPath(payload map[string]any, path string, value any) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	parts := strings.Split(path, ".")
	current := payload
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func (r *AppRuntime) runREST(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "workflow": r.workflow.Name})
	})
	mux.HandleFunc(r.config.Trigger.REST.Path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload, err := decodePayload(req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		headers := map[string]string{"trigger": "rest"}
		for key, values := range req.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		if r.config.Trigger.Mode == "async" {
			messageID, correlationID := r.prepareAsyncIdentifiers(payload)
			go func() {
				processingCtx := context.WithoutCancel(req.Context())
				msg, err := r.processPayload(processingCtx, payload, headers)
				if err != nil {
					id := messageID
					if msg != nil {
						id = msg.ID
					}
					r.logger.Error("async rest message failed",
						slog.String("message_id", id),
						slog.String("error", err.Error()),
					)
					return
				}
				r.logger.Info("async rest message processed", slog.String("message_id", msg.ID))
			}()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":         "accepted",
				"mode":           "async",
				"message_id":     messageID,
				"workflow":       r.workflow.Name,
				"correlation_id": correlationID,
			})
			return
		}

		msg, err := r.processPayload(req.Context(), payload, headers)
		if msg == nil {
			if slip.IsProcessingLocked(err) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status": "processing",
					"error":  err.Error(),
				})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		status := http.StatusOK
		if err != nil {
			status = http.StatusAccepted
		}

		w.Header().Set("Content-Type", "application/json")
		if msg.TraceID != "" {
			w.Header().Set("X-Trace-ID", msg.TraceID)
		}
		if msg.Headers["traceparent"] != "" {
			w.Header().Set("traceparent", msg.Headers["traceparent"])
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message_id":      msg.ID,
			"workflow":        r.workflow.Name,
			"correlation_id":  msg.CorrelationID,
			"trace_id":        msg.TraceID,
			"cursor":          msg.Cursor(),
			"remaining_steps": msg.RemainingSteps(),
			"history":         msg.History,
			"errors":          msg.Errors,
			"payload":         msg.Payload,
			"error":           errorString(err),
		})
	})

	server := &http.Server{Addr: r.config.Trigger.REST.Addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	r.logger.Info("rest trigger listening",
		slog.String("addr", r.config.Trigger.REST.Addr),
		slog.String("path", r.config.Trigger.REST.Path),
		slog.String("mode", r.config.Trigger.Mode),
	)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (r *AppRuntime) runKafka(ctx context.Context) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  r.config.Trigger.Kafka.Brokers,
		Topic:    r.config.Trigger.Kafka.Topic,
		GroupID:  r.config.Trigger.Kafka.GroupID,
		MinBytes: r.config.Trigger.Kafka.MinBytes,
		MaxBytes: r.config.Trigger.Kafka.MaxBytes,
	})
	defer reader.Close()

	r.logger.Info("kafka trigger listening",
		slog.String("brokers", strings.Join(r.config.Trigger.Kafka.Brokers, ",")),
		slog.String("topic", r.config.Trigger.Kafka.Topic),
		slog.String("group_id", r.config.Trigger.Kafka.GroupID),
	)

	for {
		message, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		payload, err := decodePayload(bytes.NewReader(message.Value))
		if err != nil {
			r.logger.Error("kafka payload rejected", slog.String("error", err.Error()))
			continue
		}

		headers := map[string]string{"trigger": "kafka", "kafka_topic": message.Topic}
		for _, header := range message.Headers {
			headers[header.Key] = string(header.Value)
		}
		msg, err := r.processPayload(ctx, payload, headers)
		messageID := ""
		if msg != nil {
			messageID = msg.ID
		}
		r.logger.Info("kafka message processed",
			slog.String("message_id", messageID),
			slog.Int64("offset", message.Offset),
			slog.String("error", errorString(err)),
		)
	}
}

func (r *AppRuntime) runSQS(ctx context.Context) error {
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(r.config.Trigger.SQS.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")),
	}
	if strings.TrimSpace(r.config.Trigger.SQS.Endpoint) != "" {
		loadOptions = append(loadOptions, awsconfig.WithBaseEndpoint(r.config.Trigger.SQS.Endpoint))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return err
	}
	client := sqs.NewFromConfig(awsCfg)

	r.logger.Info("sqs trigger listening",
		slog.String("queue_url", r.config.Trigger.SQS.QueueURL),
		slog.String("endpoint", r.config.Trigger.SQS.Endpoint),
	)

	for {
		output, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:              &r.config.Trigger.SQS.QueueURL,
			MaxNumberOfMessages:   r.config.Trigger.SQS.MaxMessages,
			WaitTimeSeconds:       r.config.Trigger.SQS.WaitTimeSeconds,
			VisibilityTimeout:     r.config.Trigger.SQS.VisibilityTimeout,
			MessageAttributeNames: []string{"All"},
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		for _, message := range output.Messages {
			if message.Body == nil {
				continue
			}
			payload, err := decodePayload(strings.NewReader(*message.Body))
			if err != nil {
				r.logger.Error("sqs payload rejected", slog.String("error", err.Error()))
				continue
			}

			headers := map[string]string{"trigger": "sqs"}
			for key, value := range message.MessageAttributes {
				if value.StringValue != nil {
					headers[key] = *value.StringValue
				}
			}
			msg, err := r.processPayload(ctx, payload, headers)
			if err != nil {
				messageID := ""
				if msg != nil {
					messageID = msg.ID
				}
				r.logger.Error("sqs message failed",
					slog.String("message_id", messageID),
					slog.String("error", err.Error()),
				)
				continue
			}

			_, err = client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &r.config.Trigger.SQS.QueueURL,
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				return err
			}
			r.logger.Info("sqs message processed", slog.String("message_id", msg.ID))
		}
	}
}

func (r *AppRuntime) runSNS(ctx context.Context) error {
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(r.config.Trigger.SNS.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")),
	}
	if strings.TrimSpace(r.config.Trigger.SNS.Endpoint) != "" {
		loadOptions = append(loadOptions, awsconfig.WithBaseEndpoint(r.config.Trigger.SNS.Endpoint))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return err
	}
	client := sqs.NewFromConfig(awsCfg)

	r.logger.Info("sns trigger listening",
		slog.String("subscription_queue_url", r.config.Trigger.SNS.QueueURL),
		slog.String("endpoint", r.config.Trigger.SNS.Endpoint),
	)

	for {
		output, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:              &r.config.Trigger.SNS.QueueURL,
			MaxNumberOfMessages:   r.config.Trigger.SNS.MaxMessages,
			WaitTimeSeconds:       r.config.Trigger.SNS.WaitTimeSeconds,
			VisibilityTimeout:     r.config.Trigger.SNS.VisibilityTimeout,
			MessageAttributeNames: []string{"All"},
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		for _, message := range output.Messages {
			if message.Body == nil {
				continue
			}
			payload, headers, err := decodeSNSPayload(*message.Body)
			if err != nil {
				r.logger.Error("sns payload rejected", slog.String("error", err.Error()))
				continue
			}
			for key, value := range message.MessageAttributes {
				if value.StringValue != nil {
					headers[key] = *value.StringValue
				}
			}
			msg, err := r.processPayload(ctx, payload, headers)
			if err != nil {
				messageID := ""
				if msg != nil {
					messageID = msg.ID
				}
				r.logger.Error("sns message failed",
					slog.String("message_id", messageID),
					slog.String("error", err.Error()),
				)
				continue
			}

			_, err = client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &r.config.Trigger.SNS.QueueURL,
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				return err
			}
			r.logger.Info("sns message processed", slog.String("message_id", msg.ID))
		}
	}
}

func (r *AppRuntime) prepareAsyncIdentifiers(payload map[string]any) (string, string) {
	correlationID, hasCorrelation := stringFromPath(payload, r.workflow.CorrelationIDPath)
	if !hasCorrelation {
		correlationID = newCorrelationUUID()
		setPayloadPath(payload, r.workflow.CorrelationIDPath, correlationID)
	}
	messageID, hasMessage := stringFromPath(payload, r.workflow.MessageIDPath)
	if !hasMessage {
		messageID = correlationID
		setPayloadPath(payload, r.workflow.MessageIDPath, messageID)
	}
	return messageID, correlationID
}

func decodeSNSPayload(body string) (map[string]any, map[string]string, error) {
	headers := map[string]string{"trigger": "sns"}
	var envelope struct {
		Type      string `json:"Type"`
		MessageID string `json:"MessageId"`
		TopicArn  string `json:"TopicArn"`
		Message   string `json:"Message"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err == nil && strings.TrimSpace(envelope.Message) != "" {
		payload, err := decodePayload(strings.NewReader(envelope.Message))
		if err != nil {
			return nil, nil, err
		}
		headers["sns_message_id"] = envelope.MessageID
		headers["sns_topic_arn"] = envelope.TopicArn
		headers["sns_type"] = envelope.Type
		return payload, headers, nil
	}

	payload, err := decodePayload(strings.NewReader(body))
	return payload, headers, err
}

func decodePayload(reader io.Reader) (map[string]any, error) {
	var payload map[string]any
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("invalid json payload: %w", err)
	}
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be null")
	}
	return normalizeJSONNumbers(payload).(map[string]any), nil
}

func normalizeJSONNumbers(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			typed[key] = normalizeJSONNumbers(item)
		}
		return typed
	case []any:
		for index, item := range typed {
			typed[index] = normalizeJSONNumbers(item)
		}
		return typed
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return i
		}
		f, _ := typed.Float64()
		return f
	default:
		return value
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func firstHeader(headers map[string]string, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(headers[name]); value != "" {
			return value
		}
	}
	return ""
}
