package evaluation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
)

type Score struct {
	ToolTrajectoryScore float64
	Rouge1Score         float64
}

type Result struct {
	Score    Score
	EvalFile string
}

func Evaluate(ctx context.Context, t *team.Team, evalsDir string) ([]Result, error) {
	evalFiles, err := os.ReadDir(evalsDir)
	if err != nil {
		return nil, err
	}

	var evals []session.Session
	for _, evalFile := range evalFiles {
		evalFile, err := os.ReadFile(filepath.Join(evalsDir, evalFile.Name()))
		if err != nil {
			return nil, err
		}

		var sess session.Session
		if err := json.Unmarshal(evalFile, &sess); err != nil {
			return nil, err
		}

		evals = append(evals, sess)
	}

	var results []Result
	for i := range evals {
		rt, err := runtime.New(t)
		if err != nil {
			return nil, err
		}

		actualMessages, err := runLoop(ctx, rt, &evals[i])
		if err != nil {
			return nil, err
		}

		score := score(evals[i].GetAllMessages(), actualMessages)

		results = append(results, Result{
			Score:    score,
			EvalFile: evals[i].ID,
		})
	}

	return results, nil
}

func runLoop(ctx context.Context, rt *runtime.LocalRuntime, eval *session.Session) ([]session.Message, error) {
	var userMessages []session.Message
	allMessages := eval.GetAllMessages()
	for i := range allMessages {
		if allMessages[i].Message.Role == chat.MessageRoleUser {
			userMessages = append(userMessages, allMessages[i])
		}
	}

	sess := session.New(
		session.WithToolsApproved(true),
		session.WithMaxIterations(rt.CurrentAgent().MaxIterations()),
	)
	for i := range userMessages {
		sess.AddMessage(&userMessages[i])
		_, err := rt.Run(ctx, sess)
		if err != nil {
			return nil, err
		}

		// Note: rt.Run now returns all messages, but we use sess.GetAllMessages() instead
	}

	return sess.GetAllMessages(), nil
}
