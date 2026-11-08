package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"harmony/component/config"
	"harmony/infra/model/recognize"
	"harmony/infra/utils"
)

// Image 是传给识别器的图片。
type Image struct {
	Data []byte
	MIME string
}

type Result struct {
	Model     string
	Raw       string
	Items     []recognize.ProviderItem
	Page      string
	PageTotal *float64
}

// Recognizer 把一张礼簿照片识别成候选记录，实现方负责调用具体厂商。
type Recognizer interface {
	Recognize(ctx context.Context, img Image) (*Result, error)
	Name() string
}

const recognizePrompt = `你是礼簿录入助手。图片是一页中文人情礼簿，可能旋转了 90 度，请先按文字方向理解版面。
每位来客占一行或一列，通常包含：姓名、"贺礼"等字样、大写金额（如"伍拾元""壹佰元"）、
阿拉伯数字金额（如 50）、以及字迹较小的手写还礼标注（如"11年国庆 孙女 100元 火炮20元"）。
页面角落可能有页码（如"(1-10)"）和本页合计（如"1070元"）。
按纸本顺序逐位识别，只输出 JSON，不要任何解释：
{"page":"页码或空","pageTotal":本页合计数字或null,
 "items":[{"seq":序号从1开始,"name":"","amountUpper":"大写原文","amountLower":阿拉伯数字或null,
 "amount":最终判定的元,"giftName":"","itemKind":"cash|huopao|gift","direction":"in",
 "note":"还礼标注原文，没有则空","raw":"整行原文","confidence":0到1}]}
规则：大写金额与阿拉伯数字一致时 amount 取该值；不一致时以大写为准并把 confidence 降到 0.5 以下；
看不清的字保留原样并降低 confidence；标注照抄，不要改写；礼簿默认是收礼。`

type OpenAICompatRecognizer struct {
	provider    string
	baseURL     string
	apiKey      string
	model       string
	temperature float64
	client      *http.Client
}

func NewOpenAICompatRecognizer(cfg config.RecognizeConfig) *OpenAICompatRecognizer {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &OpenAICompatRecognizer{
		provider:    cfg.Provider,
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		temperature: cfg.Temperature,
		client:      &http.Client{Timeout: timeout},
	}
}

func (r *OpenAICompatRecognizer) Name() string {
	return r.provider
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (r *OpenAICompatRecognizer) Recognize(ctx context.Context, img Image) (*Result, error) {
	if r.apiKey == "" {
		return nil, fmt.Errorf("识别服务未配置 API Key")
	}
	if r.baseURL == "" || r.model == "" {
		return nil, fmt.Errorf("识别服务未配置 baseURL 或 model")
	}
	payload := chatRequest{
		Model:       r.model,
		Temperature: r.temperature,
		Messages: []chatMessage{
			{Role: "system", Content: "你只输出 JSON，不输出 Markdown。"},
			{Role: "user", Content: []contentPart{
				{Type: "text", Text: recognizePrompt},
				{Type: "image_url", ImageURL: &imageURL{URL: "data:" + img.MIME + ";base64," + base64.StdEncoding.EncodeToString(img.Data)}},
			}},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用识别服务失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("读取识别服务响应失败: %w", err)
	}
	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("识别服务返回的不是 JSON: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(raw))
		if parsed.Error != nil && parsed.Error.Message != "" {
			message = parsed.Error.Message
		}
		if len(message) > 500 {
			message = message[:500]
		}
		return nil, fmt.Errorf("识别服务返回 %d: %s", resp.StatusCode, message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("识别服务没有返回结果")
	}
	content := parsed.Choices[0].Message.Content

	var objectResult struct {
		Page      string                     `json:"page"`
		PageTotal *float64                   `json:"pageTotal"`
		Items     []recognize.ProviderItem   `json:"items"`
	}
	if err := utils.UnmarshalLLMJSON(content, &objectResult); err != nil {
		var items []recognize.ProviderItem
		if errArray := utils.UnmarshalLLMJSON(content, &items); errArray != nil {
			return nil, fmt.Errorf("识别结果格式异常: %w", err)
		}
		objectResult.Items = items
	}
	model := parsed.Model
	if model == "" {
		model = r.model
	}
	return &Result{
		Model:     model,
		Raw:       content,
		Items:     objectResult.Items,
		Page:      objectResult.Page,
		PageTotal: objectResult.PageTotal,
	}, nil
}
