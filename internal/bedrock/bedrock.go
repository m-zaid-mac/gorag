// Package bedrock wraps the two Bedrock calls this project needs:
// Titan Embeddings V2 for vectors, and Claude for answer generation.
// Kept intentionally thin so the rest of the codebase depends on a
// small interface-shaped surface (Embed, Generate) rather than the AWS
// SDK directly.
package bedrock

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// Client talks to AWS Bedrock for embeddings and text generation.
type Client struct {
	rt *bedrockruntime.Client
}

// NewClient loads AWS credentials from the default chain (env vars,
// shared config file, or an assumed role) and returns a Bedrock client
// scoped to region.
func NewClient(ctx context.Context, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return &Client{rt: bedrockruntime.NewFromConfig(cfg)}, nil
}

type titanEmbedRequest struct {
	InputText string `json:"inputText"`
}

type titanEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Embed returns a Titan Embeddings V2 vector for the given text.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(titanEmbedRequest{InputText: text})
	if err != nil {
		return nil, err
	}

	out, err := c.rt.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String("amazon.titan-embed-text-v2:0"),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("invoking Titan embeddings: %w", err)
	}

	var resp titanEmbedResponse
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		return nil, fmt.Errorf("decoding Titan response: %w", err)
	}
	return resp.Embedding, nil
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeRequest struct {
	AnthropicVersion string          `json:"anthropic_version"`
	MaxTokens        int             `json:"max_tokens"`
	Messages         []claudeMessage `json:"messages"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// DefaultGenerateModel is a current, non-deprecated, low-cost Bedrock
// model as of writing. AWS periodically retires older Claude snapshots
// on Bedrock (this project originally shipped pointing at
// anthropic.claude-3-5-sonnet-20240620-v1:0, which has since been
// retired) and newer models are invoked via an "inference profile" ID
// — a region prefix ("us.", "global.", etc.) in front of the base model
// ID — rather than the bare model ID. If this default ever breaks,
// check AWS Console > Bedrock > Model access for what your account
// currently has enabled, and pass -model on `gorag query` to override.
const DefaultGenerateModel = "us.anthropic.claude-haiku-4-5-20251001-v1:0"

// Generate sends a prompt to Claude via Bedrock and returns the reply
// text. model is a Bedrock model ID or inference-profile ID, e.g.
// DefaultGenerateModel.
func (c *Client) Generate(ctx context.Context, model, prompt string) (string, error) {
	reqBody := claudeRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        1024,
		Messages:         []claudeMessage{{Role: "user", Content: prompt}},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	out, err := c.rt.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(model),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return "", fmt.Errorf("invoking Claude (model %q): %w", model, err)
	}

	var resp claudeResponse
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		return "", fmt.Errorf("decoding Claude response: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}
	return resp.Content[0].Text, nil
}
