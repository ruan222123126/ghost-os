use anyhow::{Result, anyhow, bail};
use rustyline::DefaultEditor;
use rustyline::error::ReadlineError;

use crate::client::BridgeClient;
use crate::types::{AgentPayload, AgentSendAwaitingHumanResponse, AskHumanOption};

use super::view;

pub(super) fn continue_awaiting_human(
    client: &BridgeClient,
    editor: &mut DefaultEditor,
    payload: &AgentSendAwaitingHumanResponse,
) -> Result<AgentPayload> {
    let question = PendingQuestion::parse(payload)?;
    view::print_pending_question(&question.prompt);
    let answer = read_pending_question_answer(editor, payload)?;
    view::print_thinking();
    submit_answer(client, &question, answer)
}

struct PendingQuestion {
    session_id: String,
    question_id: String,
    prompt: String,
}

impl PendingQuestion {
    fn parse(payload: &AgentSendAwaitingHumanResponse) -> Result<Self> {
        Ok(Self {
            session_id: required_value(payload.session_id.trim(), "session_id")?,
            question_id: required_value(payload.question_id.trim(), "question_id")?,
            prompt: required_value(payload.prompt.trim(), "prompt")?,
        })
    }
}

enum PendingQuestionAnswer {
    Answered(String),
    Cancelled,
}

fn required_value(value: &str, field: &str) -> Result<String> {
    if value.is_empty() {
        bail!("awaiting_human payload missing {field}");
    }
    Ok(value.to_string())
}

fn submit_answer(
    client: &BridgeClient,
    question: &PendingQuestion,
    answer: PendingQuestionAnswer,
) -> Result<AgentPayload> {
    match answer {
        PendingQuestionAnswer::Answered(answer) => {
            client.answer_question(&question.session_id, &question.question_id, &answer)
        }
        PendingQuestionAnswer::Cancelled => {
            client.cancel_question(&question.session_id, &question.question_id)
        }
    }
}

fn read_pending_question_answer(
    editor: &mut DefaultEditor,
    payload: &AgentSendAwaitingHumanResponse,
) -> Result<PendingQuestionAnswer> {
    match payload.options.as_deref() {
        Some(options) if !options.is_empty() => {
            let selection_mode = payload.selection_mode.as_deref().unwrap_or("single");
            read_choice_answer(editor, selection_mode, options)
        }
        _ => read_freeform_answer(editor),
    }
}

fn read_freeform_answer(editor: &mut DefaultEditor) -> Result<PendingQuestionAnswer> {
    match editor.readline(&view::answer_prompt()) {
        Ok(answer) => parse_freeform_answer(&answer),
        Err(ReadlineError::Interrupted | ReadlineError::Eof) => {
            Ok(PendingQuestionAnswer::Cancelled)
        }
        Err(err) => Err(err.into()),
    }
}

fn parse_freeform_answer(answer: &str) -> Result<PendingQuestionAnswer> {
    let answer = answer.trim();
    if answer.eq_ignore_ascii_case("/cancel") {
        return Ok(PendingQuestionAnswer::Cancelled);
    }
    if answer.is_empty() {
        bail!("answer cannot be empty");
    }
    Ok(PendingQuestionAnswer::Answered(answer.to_string()))
}

fn read_choice_answer(
    editor: &mut DefaultEditor,
    selection_mode: &str,
    options: &[AskHumanOption],
) -> Result<PendingQuestionAnswer> {
    let multiple = selection_mode.eq_ignore_ascii_case("multiple");
    view::print_pending_options(options, multiple);

    let raw_selection = read_choice_input(editor, multiple)?;
    if raw_selection.trim().eq_ignore_ascii_case("/cancel") {
        return Ok(PendingQuestionAnswer::Cancelled);
    }

    let selected = parse_choice_selection(raw_selection.trim(), options.len(), multiple)?;
    let custom_text = read_custom_text(editor, options, &selected)?;
    if custom_text.as_deref() == Some("/cancel") {
        return Ok(PendingQuestionAnswer::Cancelled);
    }
    let answer = build_choice_answer(options, &selected, custom_text.as_deref().unwrap_or(""));
    if answer.is_empty() {
        bail!("answer cannot be empty");
    }
    Ok(PendingQuestionAnswer::Answered(answer))
}

fn read_choice_input(editor: &mut DefaultEditor, multiple: bool) -> Result<String> {
    let prompt = view::choice_prompt(multiple);
    match editor.readline(&prompt) {
        Ok(line) => Ok(line),
        Err(ReadlineError::Interrupted | ReadlineError::Eof) => Ok("/cancel".to_string()),
        Err(err) => Err(err.into()),
    }
}

fn read_custom_text(
    editor: &mut DefaultEditor,
    options: &[AskHumanOption],
    selected: &[usize],
) -> Result<Option<String>> {
    if !requires_custom_text(options, selected) {
        return Ok(None);
    }
    match editor.readline(&view::custom_prompt()) {
        Ok(line) => parse_custom_text(&line).map(Some),
        Err(ReadlineError::Interrupted | ReadlineError::Eof) => Ok(Some("/cancel".to_string())),
        Err(err) => Err(err.into()),
    }
}

fn requires_custom_text(options: &[AskHumanOption], selected: &[usize]) -> bool {
    selected.iter().any(|index| {
        options
            .get(*index)
            .and_then(|option| option.allow_custom)
            .unwrap_or(false)
    })
}

fn parse_custom_text(line: &str) -> Result<String> {
    if line.trim().eq_ignore_ascii_case("/cancel") {
        return Ok("/cancel".to_string());
    }
    let custom_text = line.trim();
    if custom_text.is_empty() {
        bail!("custom answer cannot be empty");
    }
    Ok(custom_text.to_string())
}

fn parse_choice_selection(
    input: &str,
    option_count: usize,
    allow_multiple: bool,
) -> Result<Vec<usize>> {
    if option_count == 0 {
        bail!("options are required");
    }

    let mut selections = Vec::new();
    for part in input
        .split(',')
        .map(str::trim)
        .filter(|part| !part.is_empty())
    {
        let parsed: usize = part
            .parse()
            .map_err(|_| anyhow!("invalid choice {part:?}; use option numbers"))?;
        if parsed == 0 || parsed > option_count {
            bail!("choice {parsed} is out of range 1..={option_count}");
        }
        let index = parsed - 1;
        if !selections.contains(&index) {
            selections.push(index);
        }
    }

    if selections.is_empty() {
        bail!("selection cannot be empty");
    }
    if !allow_multiple && selections.len() != 1 {
        bail!("choose exactly one option");
    }
    Ok(selections)
}

fn build_choice_answer(
    options: &[AskHumanOption],
    selected: &[usize],
    custom_text: &str,
) -> String {
    let custom_text = custom_text.trim();
    if custom_text.eq_ignore_ascii_case("/cancel") {
        return String::new();
    }

    let mut parts = Vec::new();
    for index in selected {
        let Some(option) = options.get(*index) else {
            continue;
        };
        if option.allow_custom.unwrap_or(false) {
            if !custom_text.is_empty() {
                parts.push(custom_text.to_string());
            }
            continue;
        }

        let label = option.label.trim();
        if !label.is_empty() {
            parts.push(label.to_string());
        }
    }

    match parts.len() {
        0 => String::new(),
        1 => parts.remove(0),
        _ => format!("- {}", parts.join("\n- ")),
    }
}

#[cfg(test)]
mod tests {
    use super::{AskHumanOption, build_choice_answer, parse_choice_selection};

    #[test]
    fn parse_choice_selection_accepts_single_and_multiple_modes() {
        assert_eq!(parse_choice_selection("2", 3, false).unwrap(), vec![1]);
        assert_eq!(
            parse_choice_selection("1, 3, 1", 3, true).unwrap(),
            vec![0, 2]
        );
    }

    #[test]
    fn build_choice_answer_formats_multiple_selection_as_bullets() {
        let answer = build_choice_answer(
            &[
                AskHumanOption {
                    label: "Ship now".to_string(),
                    allow_custom: None,
                },
                AskHumanOption {
                    label: "Wait for QA".to_string(),
                    allow_custom: None,
                },
                AskHumanOption {
                    label: "Other".to_string(),
                    allow_custom: Some(true),
                },
            ],
            &[0, 2],
            "Custom rollout",
        );

        assert_eq!(answer, "- Ship now\n- Custom rollout");
    }
}
