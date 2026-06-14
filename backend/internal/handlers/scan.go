package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type ScanHandler struct {
	apiKey string
}

func NewScanHandler(apiKey string) *ScanHandler {
	return &ScanHandler{apiKey: apiKey}
}

type ScanResult struct {
	ProviderName string   `json:"provider_name"`
	Amount       *float64 `json:"amount"`
	DueDate      string   `json:"due_date"`
	IssuedDate   string   `json:"issued_date"`
	Notes        string   `json:"notes"`
}

func (h *ScanHandler) ScanBill(c *gin.Context) {
	if h.apiKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bill scanning is not configured"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 10<<20)) // 10 MB limit
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read file"})
		return
	}

	mediaType := detectMediaType(header.Filename, header.Header.Get("Content-Type"))
	encoded := base64.StdEncoding.EncodeToString(data)

	result, err := callClaudeVision(h.apiKey, encoded, mediaType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not analyse file: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func detectMediaType(filename, contentType string) string {
	if contentType != "" && contentType != "application/octet-stream" {
		return contentType
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	default:
		return "image/jpeg"
	}
}

func callClaudeVision(apiKey, base64Data, mediaType string) (*ScanResult, error) {
	var contentItem map[string]interface{}

	if mediaType == "application/pdf" {
		contentItem = map[string]interface{}{
			"type": "document",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": "application/pdf",
				"data":       base64Data,
			},
		}
	} else {
		contentItem = map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": mediaType,
				"data":       base64Data,
			},
		}
	}

	requestBody := map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 512,
		"system":     "You extract billing information from documents and images. Return ONLY valid JSON, no markdown, no explanation.",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []interface{}{
					contentItem,
					map[string]interface{}{
						"type": "text",
						"text": `Extract billing information from this document. Return ONLY this JSON (null for any field you cannot find):
{"provider_name":"company or issuer name","amount":0.00,"due_date":"YYYY-MM-DD","issued_date":"YYYY-MM-DD","notes":"brief note about the bill"}`,
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claude API error %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, err
	}
	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	text := strings.TrimSpace(claudeResp.Content[0].Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result ScanResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("could not parse response: %s", text)
	}

	return &result, nil
}
