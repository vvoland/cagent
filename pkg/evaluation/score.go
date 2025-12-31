package evaluation

import (
	"strings"

	"github.com/docker/cagent/pkg/session"
)

func score(expectedMessages, actualMessages []session.Message) Score {
	var expectedToolMessages []session.Message
	for i := range expectedMessages {
		if len(expectedMessages[i].Message.ToolCalls) > 0 {
			expectedToolMessages = append(expectedToolMessages, expectedMessages[i])
		}
	}

	var actualToolMessages []session.Message
	for i := range actualMessages {
		if len(actualMessages[i].Message.ToolCalls) > 0 {
			actualToolMessages = append(actualToolMessages, actualMessages[i])
		}
	}

	toolTrajectoryScore := toolTrajectoryScore(expectedToolMessages, actualToolMessages)
	rouge1Score := rouge1(expectedMessages[len(expectedMessages)-1].Message.Content, actualMessages[len(actualMessages)-1].Message.Content)

	return Score{
		ToolTrajectoryScore: toolTrajectoryScore,
		Rouge1Score:         rouge1Score,
	}
}

// https://medium.com/nlplanet/two-minutes-nlp-learn-the-rouge-metric-by-examples-f179cc285499
func rouge1(expected, actual string) float64 {
	if expected == "" && actual == "" {
		return 1.0
	}
	if expected == "" || actual == "" {
		return 0.0
	}

	expectedWords := strings.Fields(strings.ToLower(expected))
	actualWords := strings.Fields(strings.ToLower(actual))

	expectedSet := make(map[string]int)
	for _, word := range expectedWords {
		expectedSet[word]++
	}

	actualSet := make(map[string]int)
	for _, word := range actualWords {
		actualSet[word]++
	}

	overlap := 0
	for word, expectedCount := range expectedSet {
		if actualCount, exists := actualSet[word]; exists {
			overlap += min(actualCount, expectedCount)
		}
	}

	precision := float64(overlap) / float64(len(actualWords))
	recall := float64(overlap) / float64(len(expectedWords))

	if precision+recall == 0 {
		return 0.0
	}

	return 2 * (precision * recall) / (precision + recall)
}

func toolTrajectoryScore(expectedToolMessages, actualToolMessages []session.Message) float64 {
	countExpectedToolCalls := 0
	for _, m := range expectedToolMessages {
		countExpectedToolCalls += len(m.Message.ToolCalls)
	}

	countActualToolCalls := 0
	for _, m := range actualToolMessages {
		countActualToolCalls += len(m.Message.ToolCalls)
	}

	maximum := max(countExpectedToolCalls, countActualToolCalls)
	if maximum == 0 {
		return 1.0
	}

	score := 0.0
	for i := range min(len(expectedToolMessages), len(actualToolMessages)) {
		expected := expectedToolMessages[i]
		actual := actualToolMessages[i]

		for j := range expected.Message.ToolCalls {
			if j >= len(actual.Message.ToolCalls) {
				continue
			}

			if expected.Message.ToolCalls[j].Function.Name == actual.Message.ToolCalls[j].Function.Name {
				score += 1.0
			}
		}
	}

	return score / float64(maximum)
}
