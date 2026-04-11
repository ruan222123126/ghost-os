package orchestration

import (
	"context"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgerss "ghost-os/bridge/rss"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

var defaultRSSReportToolScope = []string{
	"script_exec",
	"read_and_summarize",
	"web_search",
	"rss_fetch",
}

type runtimeRSSReportRunner struct {
	runtimeStore   *bridgeruntime.ConfigStore
	runtimeFactory bridgeruntime.AgentRuntimeFactory
	allowedTools   []string
}

func newRuntimeRSSReportBuilder(store bridgeconfig.Store) bridgerss.RSSReportBuilder {
	runner := runtimeRSSReportRunner{
		runtimeStore:   bridgeruntime.WrapConfigStore(store),
		runtimeFactory: bridgeruntime.NewAgentRuntimeFactory(),
		allowedTools:   append([]string(nil), defaultRSSReportToolScope...),
	}
	return bridgerss.NewRSSReportBuilder(runner.run, bridgerss.DefaultRSSReportTimeout)
}

func (r runtimeRSSReportRunner) run(ctx context.Context, input bridgerss.RSSReportBuildInput) (string, error) {
	deps, err := r.runtimeFactory.Build(r.runtimeStore)
	if err != nil {
		return "", err
	}
	defer deps.Close()
	scoped := tools.NewScopedCatalog(deps.Registry(), append([]string(nil), r.allowedTools...))
	basePrompt, err := bridgeruntime.BuildSystemPromptForCatalog(deps.Config(), scoped)
	if err != nil {
		return "", err
	}
	systemPrompt := basePrompt + "\n\n" + bridgerss.RSSReportInvestigationSystemPrompt()
	reportAgent := agent.NewAgent(deps.Client(), scoped, systemPrompt, deps.Config().MaxTurns)
	reportAgent.SetResponseOptions(llm.CloneResponseOptions(deps.Config().ResponseOptions))
	toolGuidance := bridgerss.RenderRSSReportToolGuidance(tools.CatalogToolNames(scoped))
	return reportAgent.RunWithTraceID(
		ctx,
		bridgerss.RenderAgentRSSReportPrompt(
			input.Report,
			input.Briefing,
			input.Groups,
			input.Query,
			toolGuidance,
		),
		input.Query.TraceID,
	)
}
