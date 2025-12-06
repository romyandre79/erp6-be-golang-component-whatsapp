package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Input struct {
	Params []struct {
		InputName string `json:"inputname"`
		CompValue string `json:"compvalue"`
	} `json:"params"`
}

type Output struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
}

type WhatsAppMessage struct {
	MessagingProduct string    `json:"messaging_product"`
	To               string    `json:"to"`
	Type             string    `json:"type"`
	Text             *TextBody `json:"text,omitempty"`
	Template         *Template `json:"template,omitempty"`
}

type TextBody struct {
	Body string `json:"body"`
}

type Template struct {
	Name     string           `json:"name"`
	Language TemplateLanguage `json:"language"`
}

type TemplateLanguage struct {
	Code string `json:"code"`
}

func main() {
	var input Input
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		json.NewEncoder(os.Stdout).Encode(Output{Error: fmt.Sprintf("failed to decode input: %v", err)})
		return
	}

	var (
		token         string
		phoneNumberID string
		action        = "send_text"
		to            string
		message       string // text body or template name
		language      = "en_US"
	)

	// Extract parameters
	for _, p := range input.Params {
		val := strings.TrimSpace(p.CompValue)
		switch strings.ToLower(p.InputName) {
		case "token":
			token = val
		case "phone_number_id":
			phoneNumberID = val
		case "action":
			if val != "" {
				action = strings.ToLower(val)
			}
		case "to":
			to = val
		case "message":
			message = val
		case "language":
			if val != "" {
				language = val
			}
		}
	}

	// Validate required parameters
	if token == "" {
		json.NewEncoder(os.Stdout).Encode(Output{Error: "token is required"})
		return
	}
	if phoneNumberID == "" {
		json.NewEncoder(os.Stdout).Encode(Output{Error: "phone_number_id is required"})
		return
	}
	if to == "" {
		json.NewEncoder(os.Stdout).Encode(Output{Error: "to is required"})
		return
	}
	if message == "" {
		json.NewEncoder(os.Stdout).Encode(Output{Error: "message (text or template name) is required"})
		return
	}

	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", phoneNumberID)

	var payload WhatsAppMessage
	payload.MessagingProduct = "whatsapp"
	payload.To = to

	switch action {
	case "send_text":
		payload.Type = "text"
		payload.Text = &TextBody{Body: message}
	case "send_template":
		payload.Type = "template"
		payload.Template = &Template{
			Name: message,
			Language: TemplateLanguage{
				Code: language,
			},
		}
	default:
		json.NewEncoder(os.Stdout).Encode(Output{Error: "invalid action"})
		return
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		json.NewEncoder(os.Stdout).Encode(Output{Error: fmt.Sprintf("failed to marshal payload: %v", err)})
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		json.NewEncoder(os.Stdout).Encode(Output{Error: fmt.Sprintf("failed to create request: %v", err)})
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		json.NewEncoder(os.Stdout).Encode(Output{Error: fmt.Sprintf("request failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// If not JSON, return as string
		result = string(body)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		json.NewEncoder(os.Stdout).Encode(Output{Result: result})
	} else {
		json.NewEncoder(os.Stdout).Encode(Output{Error: fmt.Sprintf("API error: %s", string(body))})
	}
}
