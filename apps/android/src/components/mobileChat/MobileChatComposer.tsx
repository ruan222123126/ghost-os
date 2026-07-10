import { forwardRef, memo, useEffect, useImperativeHandle, useRef, useState } from "react";
import type { AgentModeSelection, ChatSelectedSkill, SkillPayload } from "../../mobileTypes";
import { ChatComposer } from "./ChatComposer";

export interface MobileChatComposerHandle {
  reset: () => void;
  setDraft: (value: string) => void;
}

interface MobileChatComposerProps {
  agentMode: AgentModeSelection;
  canEnableCodexMode: boolean;
  canSend: boolean;
  canStop: boolean;
  loading: boolean;
  skills?: SkillPayload[];
  onChangeAgentMode: (mode: AgentModeSelection) => void;
  onRefreshSkills?: () => Promise<boolean> | Promise<void> | boolean | void;
  onSend: (message: string, selectedSkill?: ChatSelectedSkill) => Promise<boolean>;
  onStop?: () => Promise<boolean | void>;
}

const MobileChatComposerBase = forwardRef<MobileChatComposerHandle, MobileChatComposerProps>(
  function MobileChatComposer(props, ref) {
    const [draft, setDraft] = useState("");
    const [selectedSkill, setSelectedSkill] = useState<ChatSelectedSkill | null>(null);
    const resetVersionRef = useRef(0);
    const skillsEnabled = props.skills !== undefined;
    const canSubmit = draft.trim().length > 0 || selectedSkill !== null;

    useImperativeHandle(ref, () => ({
      reset: () => {
        resetVersionRef.current += 1;
        setDraft("");
        setSelectedSkill(null);
      },
      setDraft,
    }), []);

    useEffect(() => {
      if (!skillsEnabled && selectedSkill !== null) {
        setSelectedSkill(null);
      }
    }, [selectedSkill, skillsEnabled]);

    async function submit(): Promise<void> {
      const message = draft.trim();
      const skill = selectedSkill ?? undefined;
      if (!message && !skill) {
        return;
      }

      const previousDraft = draft;
      const resetVersion = resetVersionRef.current;
      setDraft("");
      setSelectedSkill(null);
      const sent = await props.onSend(message, skill);
      if (!sent && resetVersionRef.current === resetVersion) {
        setDraft((current) => current || previousDraft);
        setSelectedSkill((current) => current ?? skill ?? null);
      }
    }

    return (
      <ChatComposer
        agentMode={props.agentMode}
        canEnableCodexMode={props.canEnableCodexMode}
        canSubmit={canSubmit}
        canStop={props.canStop}
        disabled={!props.canSend}
        loading={props.loading}
        selectedSkill={selectedSkill}
        skills={props.skills}
        value={draft}
        onChange={setDraft}
        onChangeAgentMode={props.onChangeAgentMode}
        onClearSelectedSkill={() => setSelectedSkill(null)}
        onRefreshSkills={props.onRefreshSkills}
        onSelectSkill={skillsEnabled ? (skill) => setSelectedSkill({ id: skill.id, name: skill.name }) : undefined}
        onStop={props.onStop ? async () => {
          await props.onStop?.();
        } : undefined}
        onSubmit={async (event) => {
          event.preventDefault();
          await submit();
        }}
      />
    );
  },
);

export const MobileChatComposer = memo(MobileChatComposerBase);
