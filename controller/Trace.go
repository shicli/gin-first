package controller

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/credentials"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	ServiceName  = getEnv("SERVICE_NAME", "test")
	collectorURL = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317") // Assuming localhost as default
	insecure     = getEnv("INSECURE", "true")
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func InitTrace() (func(context.Context) error, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 定义 grpc 连接是否采用安全模式
	secureOption := otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	if insecure == "true" || insecure == "1" {
		secureOption = otlptracegrpc.WithInsecure()
	}
	// 创建 otlp exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(collectorURL),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating otlp exporter: %w", err)
	}

	// 设置 resource
	resources, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			// 服务名称
			semconv.ServiceName(ServiceName),
			// 版本
			semconv.ServiceVersion("v0.1.0"),
			// 自定义数据
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		return nil, err
	}

	// 创建TracerProvider
	otlpProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		// 默认为 5s,为便于演示,设置为 1s
		trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second)),
		trace.WithResource(resources),
	)

	// 设置全局TracerProvider
	otel.SetTracerProvider(otlpProvider)

	// 设置传播上下文的处理器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return exporter.Shutdown, nil
}

func testSpan(c *gin.Context) {

	// 从Gin请求中获取trace上下文
	ctx := c.Request.Context()

	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	// 构造请求到 /api/v1/users
	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:8081/version", nil)
	//req, err := http.NewRequest("GET", "http://127.0.0.1:8081/version", nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to send request"})
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response body"})
		return
	}

	// 将响应返回给原始请求的客户端
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
