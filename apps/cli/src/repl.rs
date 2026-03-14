use anyhow::{Result, bail};
use colored::Colorize;
use rustyline::DefaultEditor;
use rustyline::error::ReadlineError;

use crate::client::BridgeClient;
use crate::commands::{CommandAction, handle_command};
use crate::types::{AgentPayload, AskHumanOption};

pub struct Repl {
    client: BridgeClient,
    editor: DefaultEditor,
    current_session_id: Option<String>,
}

impl Repl {
    pub fn new(client: BridgeClient) -> Result<Self> {
        let editor = DefaultEditor::new()?;
        Ok(Self {
            client,
            editor,
            current_session_id: None,
        })
    }

    pub fn run(&mut self) -> Result<()> {
        self.print_welcome();
        let prompt = "ghost> ".bright_black().to_string();

        loop {
            match self.editor.readline(&prompt) {
                Ok(line) => {
                    let input = line.trim();
                    if input.is_empty() {
                        continue;
                    }
                    let _ = self.editor.add_history_entry(input);

                    match route_input(input) {
                        RoutedInput::Command(command) => match handle_command(
                            command,
                            &self.client,
                            &mut self.current_session_id,
                        ) {
                            Ok(CommandAction::Continue) => {}
                            Ok(CommandAction::Exit) => {
                                println!("{}", "Goodbye.".dimmed());
                                break;
                            }
                            Err(err) => print_error(&err.to_string()),
                        },
                        RoutedInput::Message(message) => {
                            println!("{}", "[Agent is thinking...]".bright_black());
                            match self
                                .client
                                .send_message(message, self.current_session_id.as_deref())
                            {
                                Ok(payload) => {
                                    if let Err(err) = self.handle_agent_payload(payload) {
                                        print_error(&err.to_string());
                                    }
                                }
                                Err(err) => print_error(&err.to_string()),
                            }
                        }
                    }
                }
                Err(ReadlineError::Interrupted) => {
                    println!(
                        "{}",
                        "Input cancelled. Press Ctrl-D or /exit to quit.".dimmed()
                    );
                }
                Err(ReadlineError::Eof) => {
                    println!("{}", "Goodbye.".dimmed());
                    break;
                }
                Err(err) => return Err(err.into()),
            }
        }

        Ok(())
    }

    fn print_welcome(&self) {
        println!(
            "{}",
            format!("Ghost-OS CLI v{}", env!("CARGO_PKG_VERSION")).white()
        );
        println!(
            "{} {}",
            "Connected to bridge at".dimmed(),
            self.client.bridge_url().white()
        );
        println!();
        println!("{}", "Type your message to chat with the agent.".dimmed());
        println!(
            "{}",
            "Type /help for available commands, /exit to quit.".dimmed()
        );
        println!("{}", "Type /session to view current session id.".dimmed());
        println!(
            "{}",
            "Type /new-session to start a fresh conversation.".dimmed()
        );
        println!();
    }

    fn handle_agent_payload(&mut self, payload: AgentPayload) -> Result<()> {
        let mut current = payload;
        loop {
            let session_id = current.session_id().trim();
            if !session_id.is_empty() {
                self.current_session_id = Some(session_id.to_string());
            }

            if current.as_awaiting_human().is_some() {
                // ask_human 可能连续触发多轮，这里循环处理直到拿到最终 assistant 回复。
                current = self.handle_pending_question(&current)?;
                continue;
            }

            if let Some(reply) = current.into_success() {
                println!();
                println!("{}", reply.message.white());
                println!();
            }
            return Ok(());
        }
    }

    fn handle_pending_question(&mut self, payload: &AgentPayload) -> Result<AgentPayload> {
        let Some(payload) = payload.as_awaiting_human() else {
            bail!("expected awaiting_human payload");
        };

        let session_id = payload.session_id.trim();
        if session_id.is_empty() {
            bail!("awaiting_human payload missing session_id");
        }

        let question_id = payload.question_id.trim();
        if question_id.is_empty() {
            bail!("awaiting_human payload missing question_id");
        }

        let prompt = payload.prompt.trim();
        if prompt.is_empty() {
            bail!("awaiting_human payload missing prompt");
        }

        println!();
        println!("{}", "Agent is asking for your input:".yellow());
        println!("{}", prompt.white());
        println!();

        let answer = self.read_pending_question_answer(payload)?;

        println!("{}", "[Agent is thinking...]".bright_black());
        match answer {
            PendingQuestionAnswer::Answered(answer) => {
                self.client
                    .answer_question(session_id, question_id, &answer)
            }
            PendingQuestionAnswer::Cancelled => {
                self.client.cancel_question(session_id, question_id)
            }
        }
    }

    fn read_pending_question_answer(
        &mut self,
        payload: &crate::types::AgentSendAwaitingHumanResponse,
    ) -> Result<PendingQuestionAnswer> {
        match payload.options.as_deref() {
            Some(options) if !options.is_empty() => self.read_choice_answer(
                payload.selection_mode.as_deref().unwrap_or("single"),
                options,
            ),
            _ => self.read_freeform_answer(),
        }
    }

    fn read_freeform_answer(&mut self) -> Result<PendingQuestionAnswer> {
        let answer_prompt = "Your answer (/cancel)> ".bright_yellow().to_string();
        match self.editor.readline(&answer_prompt) {
            Ok(answer) => {
                let answer = answer.trim();
                if answer.eq_ignore_ascii_case("/cancel") {
                    return Ok(PendingQuestionAnswer::Cancelled);
                }
                if answer.is_empty() {
                    bail!("answer cannot be empty");
                }
                Ok(PendingQuestionAnswer::Answered(answer.to_string()))
            }
            Err(ReadlineError::Interrupted | ReadlineError::Eof) => {
                Ok(PendingQuestionAnswer::Cancelled)
            }
            Err(err) => Err(err.into()),
        }
    }

    fn read_choice_answer(
        &mut self,
        selection_mode: &str,
        options: &[AskHumanOption],
    ) -> Result<PendingQuestionAnswer> {
        let multiple = selection_mode.eq_ignore_ascii_case("multiple");

        println!("{}", "Options:".bright_yellow());
        for (index, option) in options.iter().enumerate() {
            let suffix = if option.allow_custom.unwrap_or(false) {
                " (custom)"
            } else {
                ""
            };
            println!(
                "{} {}{}",
                format!("{:>2}.", index + 1).bright_black(),
                option.label.white(),
                suffix.bright_black(),
            );
        }
        println!(
            "{}",
            if multiple {
                "Select one or more numbers separated by commas, or /cancel.".dimmed()
            } else {
                "Select one number, or /cancel.".dimmed()
            }
        );

        let selection_prompt = if multiple {
            "Choice(s)> ".bright_yellow().to_string()
        } else {
            "Choice> ".bright_yellow().to_string()
        };
        let raw_selection = match self.editor.readline(&selection_prompt) {
            Ok(line) => line,
            Err(ReadlineError::Interrupted | ReadlineError::Eof) => {
                return Ok(PendingQuestionAnswer::Cancelled);
            }
            Err(err) => return Err(err.into()),
        };
        if raw_selection.trim().eq_ignore_ascii_case("/cancel") {
            return Ok(PendingQuestionAnswer::Cancelled);
        }

        let selected = parse_choice_selection(raw_selection.trim(), options.len(), multiple)?;
        let custom_required = selected.iter().any(|index| {
            options
                .get(*index)
                .and_then(|option| option.allow_custom)
                .unwrap_or(false)
        });

        let mut custom_text = String::new();
        if custom_required {
            let custom_prompt = "Custom answer (/cancel)> ".bright_yellow().to_string();
            match self.editor.readline(&custom_prompt) {
                Ok(line) => {
                    if line.trim().eq_ignore_ascii_case("/cancel") {
                        return Ok(PendingQuestionAnswer::Cancelled);
                    }
                    custom_text = line.trim().to_string();
                    if custom_text.is_empty() {
                        bail!("custom answer cannot be empty");
                    }
                }
                Err(ReadlineError::Interrupted | ReadlineError::Eof) => {
                    return Ok(PendingQuestionAnswer::Cancelled);
                }
                Err(err) => return Err(err.into()),
            }
        }

        let answer = build_choice_answer(options, &selected, &custom_text);
        if answer.is_empty() {
            bail!("answer cannot be empty");
        }
        Ok(PendingQuestionAnswer::Answered(answer))
    }
}

enum PendingQuestionAnswer {
    Answered(String),
    Cancelled,
}

#[derive(Debug, PartialEq, Eq)]
enum RoutedInput<'a> {
    Command(&'a str),
    Message(&'a str),
}

fn route_input(input: &str) -> RoutedInput<'_> {
    if input.starts_with("//") {
        // `//` 作为转义前缀：允许发送以 `/` 开头的普通文本而不是命令。
        let message = &input[1..];
        if !message.trim().is_empty() {
            return RoutedInput::Message(message);
        }
    }
    if input.starts_with('/') {
        return RoutedInput::Command(input);
    }
    RoutedInput::Message(input)
}

fn print_error(message: &str) {
    println!();
    println!("{}", format!("Error: {message}").red());
    println!();
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
            .map_err(|_| anyhow::anyhow!("invalid choice {part:?}; use option numbers"))?;
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
    use super::{
        AskHumanOption, RoutedInput, build_choice_answer, parse_choice_selection, route_input,
    };

    #[test]
    fn unknown_single_slash_input_routes_to_command_handler() {
        assert_eq!(route_input("/nope"), RoutedInput::Command("/nope"));
    }

    #[test]
    fn double_slash_help_routes_as_literal_message() {
        assert_eq!(route_input("//help"), RoutedInput::Message("/help"));
    }

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
