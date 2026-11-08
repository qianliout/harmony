package service

import (
	"bytes"
	"errors"
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
阿拉伯数字金额（如 50）、以及字迹较小的手写回礼标注（如"11年国庆 孙女 100元 火炮20元"）。
页面角落可能有页码（如"(1-10)"）和本页合计（如"1070元"）。
按纸本顺序逐位识别，只输出 JSON，不要任何解释：
{"page":"页码或空","pageTotal":本页合计数字或null,
 "items":[{"seq":序号从1开始,"name":"","amountUpper":"大写原文","amountLower":阿拉伯数字或null,
 "amount":最终判定的元,"giftName":"","itemKind":"cash|huopao|gift","direction":"in",
 "note":"回礼标注原文，没有则空","raw":"整行原文","confidence":0到1}]}
规则：大写金额与阿拉伯数字一致时 amount 取该值；不一致时以大写为准并把 confidence 降到 0.5 以下；
看不清的字保留原样并降低 confidence；标注照抄，不要改写；礼簿默认是收礼。
输出必须是一个 JSON 对象，第一个字符是 {，最后一个字符是 }，不要使用 Markdown 代码块。
姓名只写纸上能看到的字，看不清的字用 □ 代替，不要猜测成常见名字。
一行里出现两个金额时，按"大写在前、小写在后"的书写习惯分别填入 amountUpper 和 amountLower。`

type OpenAICompatRecognizer struct {
	provider       string
	baseURL        string
	apiKey         string
	model          string
	temperature    float64
	highResolution bool
	client         *http.Client
}

func NewOpenAICompatRecognizer(cfg config.RecognizeConfig) *OpenAICompatRecognizer {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	return &OpenAICompatRecognizer{
		provider:       cfg.Provider,
		baseURL:        strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:         cfg.APIKey,
		model:          cfg.Model,
		temperature:    cfg.Temperature,
		highResolution: cfg.HighResolution,
		client:         &http.Client{Timeout: timeout},
	}
}

func (r *OpenAICompatRecognizer) Name() string {
	return r.provider
}

type chatRequest struct {
	Model                  string        `json:"model"`
	Messages               []chatMessage `json:"messages"`
	Temperature            float64       `json:"temperature"`
	MaxTokens              int           `json:"max_tokens"`
	VLHighResolutionImages *bool         `json:"vl_high_resolution_images,omitempty"`
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
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type providerError struct {
	status  int
	message string
}

func (e *providerError) Error() string {
	return e.message
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
		MaxTokens:   4096,
		Messages: []chatMessage{
			{Role: "system", Content: "你只输出 JSON，不输出 Markdown。"},
			{Role: "user", Content: []contentPart{
				{Type: "text", Text: recognizePrompt},
				{Type: "image_url", ImageURL: &imageURL{URL: "data:" + img.MIME + ";base64," + base64.StdEncoding.EncodeToString(img.Data)}},
			}},
		},
	}
	if r.highResolution {
		high := true
		payload.VLHighResolutionImages = &high
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	for attempt := 0; ; attempt++ {
		result, err := r.call(ctx, body)
		if err == nil {
			return result, nil
		}
		wait, retry := retryDelay(err, attempt)
		if !retry {
			return nil, err
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// retryDelay 按文档的重试策略返回等待时间：429 重试两次（3s、9s），5xx/网络错误重试一次（1s）。
func retryDelay(err error, attempt int) (time.Duration, bool) {
	var providerErr *providerError
	if errors.As(err, &providerErr) {
		switch {
		case providerErr.status == http.StatusTooManyRequests:
			if attempt == 0 {
				return 3 * time.Second, true
			}
			if attempt == 1 {
				return 9 * time.Second, true
			}
			return 0, false
		case providerErr.status >= 500:
			if attempt == 0 {
				return time.Second, true
			}
			return 0, false
		default:
			return 0, false
		}
	}
	if attempt == 0 {
		return time.Second, true
	}
	return 0, false
}

func (r *OpenAICompatRecognizer) call(ctx context.Context, body []byte) (*Result, error) {
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
	parseErr := json.Unmarshal(raw, &parsed)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(raw))
		if parseErr == nil && parsed.Error != nil && parsed.Error.Message != "" {
			message = parsed.Error.Message
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, &providerError{status: resp.StatusCode, message: "识别服务的 API Key 无效或已过期"}
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, &providerError{status: resp.StatusCode, message: "识别服务繁忙，稍后再试"}
		}
		if len(message) > 500 {
			message = message[:500]
		}
		return nil, &providerError{status: resp.StatusCode, message: fmt.Sprintf("识别服务返回 %d: %s", resp.StatusCode, message)}
	}
	if parseErr != nil {
		return nil, fmt.Errorf("识别服务返回的不是 JSON: %w", parseErr)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("识别服务没有返回结果")
	}
	choice := parsed.Choices[0]
	if choice.FinishReason == "length" {
		return nil, fmt.Errorf("这一页内容太多，请分两次拍摄")
	}
	content := choice.Message.Content

	var objectResult struct {
		Page      string                   `json:"page"`
		PageTotal *float64                 `json:"pageTotal"`
		Items     []recognize.ProviderItem `json:"items"`
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

