package evaluation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"

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
	agentSource, err := config.Resolve(agentFilename)
	if err != nil {
		return err
	}

	agents, err := teamloader.Load(ctx, agentSource, runConfig)
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
	evals, err := loadEvalSessions(ctx, evalsDir)
	if err != nil {
		return nil, err
	}

	// Each eval gets a channel; results print in order as they complete.
	chans := make([]chan Result, len(evals))
	errs, ctx := errgroup.WithContext(ctx)
	errs.SetLimit(4)

	for i := range evals {
		chans[i] = make(chan Result, 1)
		errs.Go(func() error {
			result, err := runSingleEvaluation(ctx, t, &evals[i])
			if err == nil {
				chans[i] <- result
			}
			return err
		})
	}

	var results []Result
	for _, ch := range chans {
		if result, ok := <-ch; ok {
			results = append(results, result)
			onResult(result)
		}
	}

	return results, errs.Wait()
}

// loadEvalSessions reads all evaluation session files from the given directory.
func loadEvalSessions(ctx context.Context, evalsDir string) ([]session.Session, error) {
	evalFiles, err := os.ReadDir(evalsDir)
	if err != nil {
		return nil, err
	}

	var evals []session.Session
	for _, evalFile := range evalFiles {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		data, err := os.ReadFile(filepath.Join(evalsDir, evalFile.Name()))
		if err != nil {
			return nil, err
		}

		var sess session.Session
		if err := json.Unmarshal(data, &sess); err != nil {
			return nil, err
		}

		evals = append(evals, sess)
	}

	return evals, nil
}

// runSingleEvaluation runs a single evaluation and returns the result.
func runSingleEvaluation(ctx context.Context, t *team.Team, eval *session.Session) (Result, error) {
	rt, err := runtime.New(t)
	if err != nil {
		return Result{}, err
	}

	actualMessages, err := runLoop(ctx, rt, eval)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Score:    score(eval.GetAllMessages(), actualMessages),
		EvalFile: eval.ID,
	}, nil
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
