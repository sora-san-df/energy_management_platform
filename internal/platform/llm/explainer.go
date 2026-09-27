package llm

import (
	"context"
	"errors"
	"time"

	"google.golang.org/genai"

	"github.com/sora-san-df/ai-energy-management-platform/internal/domain/analysis"
)

type Generator interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type Options struct {
	Model      string
	Timeout    time.Duration
	MaxRetries int
	Backoff    time.Duration
}

type Explainer struct {
	gen   Generator
	opts  Options
	sleep func(ctx context.Context, d time.Duration) error
}

func NewExplainer(gen Generator, opts Options) *Explainer {
	if opts.Timeout <= 0 {
		opts.Timeout = 20 * time.Second
	}
	if opts.MaxRetries < 0 {
		opts.MaxRetries = 0
	}
	if opts.Backoff <= 0 {
		opts.Backoff = 500 * time.Millisecond
	}
	return &Explainer{gen: gen, opts: opts, sleep: sleepContext}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (e *Explainer) Name() string {
	return e.opts.Model
}

func (e *Explainer) Explain(ctx context.Context, req analysis.ExplainRequest) (analysis.Explanation, error) {
	prompt, err := BuildPrompt(req)
	if err != nil {
		return analysis.Explanation{}, err
	}
	var lastErr *callError
	for attempt := 0; attempt <= e.opts.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := e.sleep(ctx, e.opts.Backoff*time.Duration(1<<(attempt-1))); err != nil {
				return analysis.Explanation{}, classify(err)
			}
		}
		callCtx, cancel := context.WithTimeout(ctx, e.opts.Timeout)
		text, err := e.gen.Generate(callCtx, systemPrompt, prompt)
		cancel()
		if err == nil {
			return ParseExplanation(text)
		}
		lastErr = classify(err)
		if ctx.Err() != nil || !lastErr.retry {
			return analysis.Explanation{}, lastErr
		}
	}
	return analysis.Explanation{}, lastErr
}

type callError struct {
	kind  error
	cause error
	retry bool
}

func (e *callError) Error() string {
	return e.kind.Error() + ": " + e.cause.Error()
}

func (e *callError) Unwrap() []error {
	return []error{e.kind, e.cause}
}

func classify(err error) *callError {
	var apiErr genai.APIError
	switch {
	case errors.As(err, &apiErr):
		switch {
		case apiErr.Code == 429:
			return &callError{kind: analysis.ErrExplainerRateLimited, cause: err, retry: true}
		case apiErr.Code == 408 || apiErr.Code == 504:
			return &callError{kind: analysis.ErrExplainerTimeout, cause: err, retry: true}
		case apiErr.Code >= 500:
			return &callError{kind: analysis.ErrExplainerUnavailable, cause: err, retry: true}
		}
		return &callError{kind: analysis.ErrExplainerUnavailable, cause: err}
	case errors.Is(err, context.DeadlineExceeded):
		return &callError{kind: analysis.ErrExplainerTimeout, cause: err, retry: true}
	case errors.Is(err, context.Canceled):
		return &callError{kind: analysis.ErrExplainerUnavailable, cause: err}
	case errors.Is(err, ErrEmptyResponse):
		return &callError{kind: analysis.ErrExplainerInvalidResponse, cause: err}
	}
	return &callError{kind: analysis.ErrExplainerUnavailable, cause: err}
}
