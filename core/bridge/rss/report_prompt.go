package rss

import (
	"fmt"
	"strings"
)

const rssReportInvestigationSystemPrompt = `You are Ghost-OS RSS report investigation agent.

You are producing an end-user markdown report from an RSS briefing dossier.

Workflow requirements:
1. Read the dossier file first.
2. Use tools when needed to validate important claims, inspect linked sources, and gather missing context.
3. Prefer source-backed claims over speculation.
4. Return Markdown only.
5. The entire report must be written in Simplified Chinese.

The final report must cover:
- What happened
- Why it matters
- Opportunities
- Risks / constraints
- What may happen next
- Concrete signals to watch
- Event sources. In the "What happened" section, every major event/highlight must end with its corresponding "出处：..." line mapped to concrete source items.

Use the exact report title provided in the prompt as the first Markdown H1.

Do not mention your tool usage, internal workflow, or chain-of-thought.`

func renderAgentRSSReportPrompt(
	report RSSReportResult,
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
	query RSSReportQuery,
	toolGuidance string,
) string {
	guidance := strings.TrimSpace(toolGuidance)
	if guidance == "" {
		guidance = "Use the currently available tools when needed to validate important claims, inspect primary sources, and add missing context."
	}
	return strings.TrimSpace(fmt.Sprintf(`Read the dossier file at %s first.

%s

Return the final report in Markdown only.
Write the entire report in Simplified Chinese.
Use this exact H1 title: %s
Inside the "## 发生了什么" section, every major event/highlight must be followed by its corresponding "出处：..." line with concrete source items and links. Do not move sources into a separate appendix section.

Report context:
- report_title: %s
- report_id: %s
- briefing_id: %s
- highlight_count: %d
- group_count: %d
- trace_id: %s
- task_id: %s`, query.DossierPath, guidance, report.Title, report.Title, report.ID, briefing.ID, len(briefing.Highlights), len(groups), query.TraceID, query.TaskID))
}
