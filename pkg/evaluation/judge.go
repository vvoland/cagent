package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/model/provider"
	"github.com/docker/cagent/pkg/model/provider/options"
)

// relevancePrompt is the prompt template for the judge model to evaluate responses.
const relevancePrompt = `You are an evaluation judge. Check if the response matches the given relevance criteria.

Response to evaluate:
<response>
%s
</response>

Criteria to check:
<criteria>
%s
</criteria>

Evaluate whether the response satisfies the criteria and respond with your judgment.`

// judgeResponseSchema defines the JSON schema for structured output from the judge model.
var judgeResponseSchema = &latest.StructuredOutput{
	Name:        "judge_response",
	Description: "Evaluation result for a relevance criterion",
	Schema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"result": map[string]any{
				"type":        "string",
				"enum":        []string{"pass", "fail"},
				"description": "Whether the response satisfies the criterion",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Brief explanation of why the criterion passed or failed",
			},
		},
		"required":             []string{"result", "reason"},
		"additionalProperties": false,
	},
	Strict: true,
}

func (r *Runner) checkRelevance(ctx context.Context, response string, criteria []string) (passed int, failed, errs []string) {
	for _, criterion := range criteria {
		if ctx.Err() != nil {
			errs = append(errs, fmt.Sprintf("context cancelled checking: %s", criterion))
			continue
		}

		pass, err := r.checkSingleRelevance(ctx, response, criterion)
		if err != nil {
			errs = append(errs, fmt.Sprintf("error checking %q: %v", criterion, err))
			continue
		}

		if pass {
			passed++
		} else {
			failed = append(failed, criterion)
		}
	}

	return passed, failed, errs
}

func (r *Runner) checkSingleRelevance(ctx context.Context, response, criterion string) (bool, error) {
	// Create a provider with structured output for this specific call
	modelCfg := r.judgeModel.BaseConfig().ModelConfig
	judgeWithSchema, err := provider.New(
		ctx,
		&modelCfg,
		r.runConfig.EnvProvider(),
		options.WithStructuredOutput(judgeResponseSchema),
	)
	if err != nil {
		return false, fmt.Errorf("creating judge provider with structured output: %w", err)
	}

	prompt := fmt.Sprintf(relevancePrompt, response, criterion)

	messages := []chat.Message{{Role: chat.MessageRoleUser, Content: prompt}}
	stream, err := judgeWithSchema.CreateChatCompletionStream(ctx, messages, nil)
	if err != nil {
		return false, fmt.Errorf("creating chat completion: %w", err)
	}
	defer stream.Close()

	var fullResponse strings.Builder
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		for _, choice := range resp.Choices {
			fullResponse.WriteString(choice.Delta.Content)
		}
	}

	return parseJudgeResponse(fullResponse.String()), nil
}

// judgeResult represents the structured response from the judge model.
type judgeResult struct {
	Result string `json:"result"`
	Reason string `json:"reason"`
}

func parseJudgeResponse(text string) bool {
	text = strings.TrimSpace(text)

	var result judgeResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		// With structured output this should not happen, but handle gracefully
		return false
	}

	return strings.EqualFold(result.Result, "pass")
}
