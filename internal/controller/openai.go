package controller

import (
	"HireMeMaybe-backend/internal/model"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// VerificationResult represents the AI's decision on company verification
type VerificationResult struct {
    ShouldVerify bool   `json:"should_verify"`
    Reasoning    string `json:"reasoning"`
    Confidence   string `json:"confidence"` // High, Medium, Low
}

// OpenAIRequest represents the request structure for OpenAI API
type OpenAIRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
    } `json:"choices"`
}

// VerifyCompanyWithAI analyzes company information and determines if it should be verified
func VerifyCompanyWithAI(company model.CompanyUser) (*VerificationResult, error) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    model := os.Getenv("OPENAI_MODEL")

    if apiKey == "" {
        return nil, fmt.Errorf("OPENAI_API_KEY is not configured")
    }

    if model == "" {
        model = "gpt-4" // Default model
    }

    // Construct the prompt with company information
    prompt := fmt.Sprintf(`You are an expert company verification analyst. Analyze the following company information and determine if this company should be verified as legitimate.
		Company Information:
		- Company Name: %s
		- Industry: %s
		- Company Size: %s (XS=Extra Small, S=Small, M=Medium, L=Large, XL=Extra Large - this is fine as provided)
		- Overview/Description: %s
		- Email Domain: %s
		- Phone: %s

		VERIFICATION CRITERIA - Evaluate ONLY based on:
		1. Is the company name legitimate and professional for the given industry?
		2. Does the email domain relate to the company name (if not a generic domain like gmail.com)?
		3. Is the overview/description professionally written, coherent, and specific?
		4. Company size is already in standard format (XS/S/M/L/XL) - accept it as-is, no need to question it
		5. Are there red flags like: generic test names ("Test Company", "Company Inc"), gibberish text, obviously fake information?

		CRITICAL: DO NOT MENTION OR CONSIDER:
		- Physical addresses (we don't have this data and don't need it)
		- Business registration numbers (not required)
		- Tax IDs (not required)
		- Official websites (not required)
		- Regulatory registrations (not required)
		- Social media profiles (not required)
		- "Limited online presence" (irrelevant)
		- "Cannot verify independently" (not your job)

		DO NOT include any of these items in your reasoning. Focus ONLY on the quality of the information provided above.

		VERIFICATION APPROACH:
		- Professional company name + detailed overview + reasonable info → Verify with High confidence
		- Professional name + decent overview → Verify with Medium confidence
		- Generic/test names, vague overview, suspicious info → Do not verify

		Examples of what TO verify:
		✓ "Acme Consulting" with detailed consulting services overview
		✓ "Jane Smith Design" with specific design services description
		✓ "TechStart Solutions" with clear SaaS product explanation

		Examples of what NOT to verify:
		✗ "Test Company" or "Company Inc" or "Example Corp"
		✗ Overview like "we do business" or "testing" or gibberish
		✗ Obviously fake or placeholder information

		Based ONLY on the company name, overview quality, and information consistency, determine:
		1. Should this company be verified? (true/false)
		2. What is your reasoning? (focus on name quality, overview detail, and professionalism)
		3. What is your confidence level? (High/Medium/Low)

		Respond ONLY with a valid JSON object in this exact format:
		{
		"should_verify": true or false,
		"reasoning": "your detailed reasoning focusing on company name quality and overview professionalism",
		"confidence": "High" or "Medium" or "Low"
		}`,
        company.Name,
        company.Industry,
        getStringValue(company.Size),
        company.Overview,
        extractEmailDomain(company.User.Email),
        getStringValue(company.User.Tel),
    )

    // Prepare the request
    requestBody := OpenAIRequest{
        Model: model,
        Messages: []Message{
            {
                Role:    "system",
                Content: "You are a company verification expert. You must respond only with valid JSON.",
            },
            {
                Role:    "user",
                Content: prompt,
            },
        },
    }

    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
    }
    defer func() {
        if cerr := resp.Body.Close(); cerr != nil {
            fmt.Printf("warning: failed to close response body: %v\n", cerr)
        }
    }()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
    }

    var openAIResp OpenAIResponse
    if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    if len(openAIResp.Choices) == 0 {
        return nil, fmt.Errorf("no response from OpenAI")
    }

    // Parse the AI's response
    content := openAIResp.Choices[0].Message.Content
    var result VerificationResult
    if err := json.Unmarshal([]byte(content), &result); err != nil {
        return nil, fmt.Errorf("failed to parse AI response: %w (response: %s)", err, content)
    }

    return &result, nil
}

// Helper function to safely get string value from pointer
func getStringValue(ptr *string) string {
    if ptr == nil {
        return "Not provided"
    }
    return *ptr
}

// Helper function to extract domain from email
func extractEmailDomain(emailPtr *string) string {
    if emailPtr == nil {
        return "Not provided"
    }
    email := *emailPtr
    for i := len(email) - 1; i >= 0; i-- {
        if email[i] == '@' {
            if i+1 < len(email) {
                return email[i+1:]
            }
            break
        }
    }
    return email
}
