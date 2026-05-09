package group

import "strings"

const (
	initialTranscriptBaseCapacity = 2
	systemTranscriptSpeaker       = "system"
	upstreamTranscriptHeader      = "上游群组 transcript:\n"
)

type TranscriptEntry struct {
	Round   int    `json:"round"`
	Speaker string `json:"speaker"`
	AgentID string `json:"agent_id,omitempty"`
	Content string `json:"content"`
}

type Transcript []TranscriptEntry

func Initial(groupNode Node, previousGroupID string, previousTranscript Transcript) Transcript {
	transcript := make(Transcript, 0, initialTranscriptBaseCapacity+len(previousTranscript))
	if groupNode.Group != nil && groupNode.Group.SharedContext != "" {
		transcript = append(transcript, TranscriptEntry{
			Round:   0,
			Speaker: systemTranscriptSpeaker,
			Content: groupNode.Group.SharedContext,
		})
	}
	return transcript.WithUpstreamTranscript(previousGroupID, previousTranscript)
}

func (t Transcript) AppendMember(round int, title string, agentID string, content string) Transcript {
	return append(t.Clone(), TranscriptEntry{
		Round:   round,
		Speaker: title,
		AgentID: agentID,
		Content: content,
	})
}

func (t Transcript) Clone() Transcript {
	if t == nil {
		return nil
	}
	out := make(Transcript, len(t))
	copy(out, t)
	return out
}

func (t Transcript) Format() string {
	lines := make([]string, 0, len(t))
	for _, entry := range t {
		lines = append(lines, strings.TrimSpace(entry.Speaker+": "+entry.Content))
	}
	return strings.Join(lines, "\n")
}

func (t Transcript) WithUpstreamTranscript(previousGroupID string, previousTranscript Transcript) Transcript {
	out := t.Clone()
	if previousGroupID == "" || len(previousTranscript) == 0 {
		return out
	}
	return append(out, TranscriptEntry{
		Round:   0,
		Speaker: systemTranscriptSpeaker,
		Content: upstreamTranscriptHeader + previousTranscript.Format(),
	})
}
