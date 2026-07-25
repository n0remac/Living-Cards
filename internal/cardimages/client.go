package cardimages

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultEndpoint  = "https://api.openai.com/v1/images/generations"
	maxResponseBytes = 64 << 20
)

type GenerateOptions struct {
	Model   string
	Prompt  string
	Size    string
	Quality string
	Format  string
}

type GenerateResult struct {
	Image     []byte
	RequestID string
}

type Client struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

func (Client) Validate(options GenerateOptions) error {
	return validateGenerateOptions(options)
}

func (c Client) Generate(ctx context.Context, options GenerateOptions) (GenerateResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return GenerateResult{}, fmt.Errorf("OPENAI_API_KEY is required")
	}
	if err := validateGenerateOptions(options); err != nil {
		return GenerateResult{}, err
	}

	payload, err := json.Marshal(struct {
		Model        string `json:"model"`
		Prompt       string `json:"prompt"`
		Size         string `json:"size"`
		Quality      string `json:"quality"`
		OutputFormat string `json:"output_format"`
		N            int    `json:"n"`
	}{
		Model:        options.Model,
		Prompt:       options.Prompt,
		Size:         options.Size,
		Quality:      options.Quality,
		OutputFormat: options.Format,
		N:            1,
	})
	if err != nil {
		return GenerateResult{}, fmt.Errorf("encode image request: %w", err)
	}

	endpoint := strings.TrimSpace(c.Endpoint)
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return GenerateResult{}, fmt.Errorf("create image request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.APIKey)
	request.Header.Set("Content-Type", "application/json")

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("generate image: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return GenerateResult{}, fmt.Errorf("read image response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return GenerateResult{}, fmt.Errorf("image response exceeds %d bytes", maxResponseBytes)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return GenerateResult{}, apiResponseError(response.StatusCode, body)
	}

	var decoded struct {
		Data []struct {
			Base64 string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return GenerateResult{}, fmt.Errorf("decode image response: %w", err)
	}
	if len(decoded.Data) == 0 || strings.TrimSpace(decoded.Data[0].Base64) == "" {
		return GenerateResult{}, fmt.Errorf("image response did not contain data[0].b64_json")
	}
	image, err := base64.StdEncoding.DecodeString(decoded.Data[0].Base64)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("decode generated image: %w", err)
	}
	if len(image) == 0 {
		return GenerateResult{}, fmt.Errorf("generated image is empty")
	}
	return GenerateResult{Image: image, RequestID: response.Header.Get("X-Request-ID")}, nil
}

func validateGenerateOptions(options GenerateOptions) error {
	switch options.Model {
	case "gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini":
	default:
		return fmt.Errorf("model must be gpt-image-2, gpt-image-1.5, gpt-image-1, or gpt-image-1-mini")
	}
	if strings.TrimSpace(options.Prompt) == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	if len(options.Prompt) > 32_000 {
		return fmt.Errorf("prompt must not exceed 32000 characters")
	}
	switch options.Quality {
	case "low", "medium", "high":
	default:
		return fmt.Errorf("quality must be low, medium, or high")
	}
	switch options.Format {
	case "png", "jpeg", "webp":
	default:
		return fmt.Errorf("format must be png, jpeg, or webp")
	}
	if err := validateSize(options.Model, options.Size); err != nil {
		return err
	}
	return nil
}

func validateSize(model, size string) error {
	size = strings.TrimSpace(size)
	if model != "gpt-image-2" {
		switch size {
		case "auto", "1024x1024", "1536x1024", "1024x1536":
			return nil
		default:
			return fmt.Errorf("size for %s must be auto, 1024x1024, 1536x1024, or 1024x1536", model)
		}
	}
	if size == "auto" {
		return nil
	}
	widthText, heightText, ok := strings.Cut(size, "x")
	if !ok {
		return fmt.Errorf("size must use WIDTHxHEIGHT")
	}
	width, widthErr := strconv.Atoi(widthText)
	height, heightErr := strconv.Atoi(heightText)
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return fmt.Errorf("size must use positive integer WIDTHxHEIGHT")
	}
	if width%16 != 0 || height%16 != 0 {
		return fmt.Errorf("gpt-image-2 width and height must be divisible by 16")
	}
	if width > 3840 || height > 3840 {
		return fmt.Errorf("gpt-image-2 width and height must not exceed 3840")
	}
	pixels := int64(width) * int64(height)
	if pixels < 655_360 || pixels > 8_294_400 {
		return fmt.Errorf("gpt-image-2 size must contain between 655360 and 8294400 pixels")
	}
	long, short := width, height
	if short > long {
		long, short = short, long
	}
	if long > short*3 {
		return fmt.Errorf("gpt-image-2 aspect ratio must not exceed 3:1")
	}
	return nil
}

func apiResponseError(status int, body []byte) error {
	var response struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    any    `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &response) == nil && strings.TrimSpace(response.Error.Message) != "" {
		if response.Error.Type != "" {
			return fmt.Errorf("OpenAI image API returned HTTP %d (%s): %s", status, response.Error.Type, response.Error.Message)
		}
		return fmt.Errorf("OpenAI image API returned HTTP %d: %s", status, response.Error.Message)
	}
	return fmt.Errorf("OpenAI image API returned HTTP %d", status)
}
