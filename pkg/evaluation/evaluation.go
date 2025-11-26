package evaluation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/docker/cagent/pkg/agentfile"
	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/teamloader"
)

type Score struct {
	ToolTrajectoryScore float64
	Rouge1Score         float64
}

type Result struct {
	Score    Score
	EvalFile string
}

type Printer interface {
	Printf(format string, a ...any)
}

func Evaluate(ctx context.Context, out Printer, agentFilename, evalsDir string, runConfig *config.RuntimeConfig) error {
	agentFilename, err := agentfile.Resolve(ctx, out, agentFilename)
	if err != nil {
		return err
	}

	agents, err := teamloader.Load(ctx, agentFilename, runConfig)
	if err != nil {
		return err
	}

	_, err = runEvaluations(ctx, agents, evalsDir, func(result Result) {
		out.Printf("Eval file: %s\n", result.EvalFile)
		out.Printf("Tool trajectory score: %f\n", result.Score.ToolTrajectoryScore)
		out.Printf("Rouge-1 score: %f\n", result.Score.Rouge1Score)
	})
	return err
}

func runEvaluations(ctx context.Context, t *team.Team, evalsDir string, onResult func(Result)) ([]Result, error) {
	evalFiles, err := os.ReadDir(evalsDir)
	if err != nil {
		return nil, err
	}

	var evals []session.Session
	for _, evalFile := range evalFiles {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

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
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		rt, err := runtime.New(t)
		if err != nil {
			return nil, err
		}

		actualMessages, err := runLoop(ctx, rt, &evals[i])
		if err != nil {
			return nil, err
		}

		score := score(evals[i].GetAllMessages(), actualMessages)
		result := Result{
			Score:    score,
			EvalFile: evals[i].ID,
		}
		onResult(result)

		results = append(results, result)
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
