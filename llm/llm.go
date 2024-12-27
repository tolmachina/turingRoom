package llm

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type OllamaClient struct {
	BaseURL string
}

type AIClient interface {
	GenerateResponse(model, prompt string) (string, error)
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{BaseURL: baseURL}
}

func (c *OllamaClient) GenerateResponse(model, prompt string) (string, error) {
	reqBody, err := json.Marshal(OllamaRequest{Model: model, Prompt: prompt})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}
