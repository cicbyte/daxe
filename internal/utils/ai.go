package utils

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cicbyte/daxe/internal/models"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// AIRequestMode AI请求模式
type AIRequestMode int

const (
	// AIRequestModeSync 同步（非流式）请求
	AIRequestModeSync AIRequestMode = iota
	// AIRequestModeStream 流式请求
	AIRequestModeStream
)

// AIRequestConfig AI请求配置
type AIRequestConfig struct {
	SystemPrompt string         // 系统提示词
	UserPrompt   string         // 用户输入内容
	Mode         AIRequestMode  // 请求模式
	Streaming    bool           // 是否启用流式输出显示
}

// AIResponse AI响应结果
type AIResponse struct {
	Content   string        // AI返回的内容
	Duration  time.Duration // 请求耗时
	Success   bool          // 是否成功
	Error     error         // 错误信息
}

// CallAI 调用AI接口的统一入口
func CallAI(ctx context.Context, config *models.AppConfig, reqConfig *AIRequestConfig) *AIResponse {
	startTime := time.Now()

	response := &AIResponse{
		Success: true,
	}

	// 创建AI模型
	model, err := createAIModel(ctx, config)
	if err != nil {
		response.Success = false
		response.Error = fmt.Errorf("创建AI模型失败: %w", err)
		response.Duration = time.Since(startTime)
		return response
	}

	// 准备消息
	messages := prepareMessages(reqConfig.SystemPrompt, reqConfig.UserPrompt)

	// 根据模式执行请求
	switch reqConfig.Mode {
	case AIRequestModeStream:
		response.Content, err = callAIStream(ctx, model, messages, reqConfig.Streaming)
	case AIRequestModeSync:
		response.Content, err = callAISync(ctx, model, messages)
	default:
		response.Success = false
		response.Error = fmt.Errorf("不支持的AI请求模式: %v", reqConfig.Mode)
		response.Duration = time.Since(startTime)
		return response
	}

	if err != nil {
		response.Success = false
		response.Error = fmt.Errorf("AI处理失败: %w", err)
	}

	response.Duration = time.Since(startTime)
	return response
}

// createAIModel 创建AI模型实例
func createAIModel(ctx context.Context, config *models.AppConfig) (*openai.ChatModel, error) {
	// 准备模型配置
	maxTokens := config.AI.MaxTokens
	temperature := float32(config.AI.Temperature)

	modelConfig := &openai.ChatModelConfig{
		BaseURL:     config.AI.BaseURL,
		APIKey:      config.AI.ApiKey,
		Timeout:     time.Duration(config.AI.Timeout) * time.Second,
		Model:       config.AI.Model,
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	}

	// 特殊处理：Ollama不需要真实的API密钥
	if strings.ToLower(config.AI.Provider) == "ollama" {
		modelConfig.APIKey = "ollama"
	}

	// 创建模型实例
	model, err := openai.NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, fmt.Errorf("创建AI模型失败: %w", err)
	}

	return model, nil
}

// prepareMessages 准备AI对话消息
func prepareMessages(systemPrompt, userPrompt string) []*schema.Message {
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: systemPrompt,
		},
		{
			Role:    schema.User,
			Content: userPrompt,
		},
	}
	return messages
}

// callAISync 同步（非流式）AI请求
func callAISync(ctx context.Context, model *openai.ChatModel, messages []*schema.Message) (string, error) {
	response, err := model.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("AI同步请求失败: %w", err)
	}
	return response.Content, nil
}

// callAIStream 流式AI请求
func callAIStream(ctx context.Context, model *openai.ChatModel, messages []*schema.Message, enableStreaming bool) (string, error) {
	reader, err := model.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("AI流式请求启动失败: %w", err)
	}
	defer reader.Close()

	var result strings.Builder

	// 实时显示流式输出
	if enableStreaming {
		fmt.Print("🤖 AI回复: ")
	}

	for {
		chunk, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("AI流式读取失败: %w", err)
		}

		// 累积结果
		result.WriteString(chunk.Content)

		// 实时显示（如果启用）
		if enableStreaming {
			fmt.Print(chunk.Content)
		}
	}

	if enableStreaming {
		fmt.Println() // 换行
	}

	return result.String(), nil
}

// CallAISync 简化的同步AI调用接口
func CallAISync(ctx context.Context, config *models.AppConfig, systemPrompt, userPrompt string) (string, error) {
	reqConfig := &AIRequestConfig{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Mode:         AIRequestModeSync,
		Streaming:    false,
	}

	response := CallAI(ctx, config, reqConfig)
	if !response.Success {
		return "", response.Error
	}

	return response.Content, nil
}

// CallAIStream 简化的流式AI调用接口
func CallAIStream(ctx context.Context, config *models.AppConfig, systemPrompt, userPrompt string, enableStreaming bool) (string, error) {
	reqConfig := &AIRequestConfig{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Mode:         AIRequestModeStream,
		Streaming:    enableStreaming,
	}

	response := CallAI(ctx, config, reqConfig)
	if !response.Success {
		return "", response.Error
	}

	return response.Content, nil
}

// ValidateAIConfig 验证AI配置是否完整
func ValidateAIConfig(config *models.AppConfig) error {
	if config.AI.BaseURL == "" {
		return fmt.Errorf("AI BaseURL不能为空")
	}
	if config.AI.Model == "" {
		return fmt.Errorf("AI Model不能为空")
	}
	if config.AI.Provider == "" {
		return fmt.Errorf("AI Provider不能为空")
	}

	// Ollama通常不需要API密钥，其他提供商需要
	if strings.ToLower(config.AI.Provider) != "ollama" && config.AI.ApiKey == "" {
		return fmt.Errorf("AI ApiKey不能为空")
	}

	if config.AI.Timeout <= 0 {
		return fmt.Errorf("AI Timeout必须大于0")
	}
	if config.AI.MaxTokens <= 0 {
		return fmt.Errorf("AI MaxTokens必须大于0")
	}

	return nil
}