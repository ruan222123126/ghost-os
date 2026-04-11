package rss

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRSSReportBuilderRunsConfiguredFunction(t *testing.T) {
	builder := newFunctionRSSReportBuilder(defaultRSSReportTimeout, func(_ context.Context, input rssReportBuildInput) (string, error) {
		if input.Report.ID != "rssr_test" {
			t.Fatalf("unexpected report id: %q", input.Report.ID)
		}
		return "```markdown\n# Investigated Report\n```", nil
	})

	out, err := builder.Build(
		context.Background(),
		RSSReportResult{ID: "rssr_test"},
		RSSBriefingResult{ID: "rssb_test"},
		nil,
		RSSReportQuery{TraceID: "trace-rss-report"},
	)
	if err != nil {
		t.Fatalf("builder returned error: %v", err)
	}
	if out != "# Investigated Report" {
		t.Fatalf("unexpected markdown: %q", out)
	}
}

func TestRSSReportBuilderReturnsErrorWhenFunctionMissing(t *testing.T) {
	builder := newFunctionRSSReportBuilder(defaultRSSReportTimeout, nil)
	_, err := builder.Build(context.Background(), RSSReportResult{}, RSSBriefingResult{}, nil, RSSReportQuery{})
	if err == nil {
		t.Fatal("expected error when build function is missing")
	}
	requireStringContains(t, err.Error(), "rss report builder is not configured")
}

func TestRSSReportBuilderPropagatesBuildError(t *testing.T) {
	expected := errors.New("builder failed")
	builder := newFunctionRSSReportBuilder(defaultRSSReportTimeout, func(context.Context, rssReportBuildInput) (string, error) {
		return "", expected
	})
	_, err := builder.Build(context.Background(), RSSReportResult{}, RSSBriefingResult{}, nil, RSSReportQuery{})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestRSSReportBuilderUsesDefaultTimeoutWhenInvalid(t *testing.T) {
	builder := newFunctionRSSReportBuilder(0, func(ctx context.Context, _ rssReportBuildInput) (string, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected context deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > defaultRSSReportTimeout {
			t.Fatalf("expected deadline within report timeout, got %v", remaining)
		}
		return "# ok", nil
	})
	_, err := builder.Build(context.Background(), RSSReportResult{}, RSSBriefingResult{}, nil, RSSReportQuery{})
	if err != nil {
		t.Fatalf("builder returned error: %v", err)
	}
}

func TestRenderRSSReportToolGuidanceFormatsNames(t *testing.T) {
	guidance := renderRSSReportToolGuidance([]string{" script_exec ", "web_search"})
	requireStringContains(t, guidance, "`script_exec`")
	requireStringContains(t, guidance, "`web_search`")
	requireStringContains(t, guidance, "Investigation tools for this run")
}
