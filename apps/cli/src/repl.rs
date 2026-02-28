use anyhow::{Result, bail};
use colored::Colorize;
use rustyline::DefaultEditor;
use rustyline::error::ReadlineError;

use crate::client::BridgeClient;
use crate::commands::{CommandAction, handle_command};
use crate::types::AgentPayload;

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
            if !current.session_id.trim().is_empty() {
                self.current_session_id = Some(current.session_id.clone());
            }

            if current.status == "awaiting_human" {
                // ask_human 可能连续触发多轮，这里循环处理直到拿到最终 assistant 回复。
                current = self.handle_pending_question(&current)?;
                continue;
            }

            println!();
            println!("{}", current.message.white());
            println!();
            return Ok(());
        }
    }

    fn handle_pending_question(&mut self, payload: &AgentPayload) -> Result<AgentPayload> {
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

        let answer_prompt = "Your answer> ".bright_yellow().to_string();
        let answer = self.editor.readline(&answer_prompt)?;
        let answer = answer.trim();
        if answer.is_empty() {
            bail!("answer cannot be empty");
        }

        self.client
            .send_human_response(session_id, question_id, answer)?;
        println!("{}", "[Agent is thinking...]".bright_black());
        self.client.send_message("", Some(session_id))
    }
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

#[cfg(test)]
mod tests {
    use super::{RoutedInput, route_input};

    #[test]
    fn unknown_single_slash_input_routes_to_command_handler() {
        assert_eq!(route_input("/nope"), RoutedInput::Command("/nope"));
    }

    #[test]
    fn double_slash_help_routes_as_literal_message() {
        assert_eq!(route_input("//help"), RoutedInput::Message("/help"));
    }
}
