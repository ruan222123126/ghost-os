use anyhow::Result;
use colored::Colorize;
use rustyline::DefaultEditor;
use rustyline::error::ReadlineError;

use crate::client::BridgeClient;
use crate::commands::{CommandAction, handle_command};

pub struct Repl {
    client: BridgeClient,
    editor: DefaultEditor,
}

impl Repl {
    pub fn new(client: BridgeClient) -> Result<Self> {
        let editor = DefaultEditor::new()?;
        Ok(Self { client, editor })
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
                        RoutedInput::Command(command) => match handle_command(command, &self.client) {
                            Ok(CommandAction::Continue) => {}
                            Ok(CommandAction::Exit) => {
                                println!("{}", "Goodbye.".dimmed());
                                break;
                            }
                            Err(err) => print_error(&err.to_string()),
                        },
                        RoutedInput::Message(message) => {
                            println!("{}", "[Agent is thinking...]".bright_black());
                            match self.client.send_message(message) {
                                Ok(reply) => {
                                    println!();
                                    println!("{}", reply.white());
                                    println!();
                                }
                                Err(err) => print_error(&err.to_string()),
                            }
                        }
                    }
                }
                Err(ReadlineError::Interrupted) => {
                    println!("{}", "Input cancelled. Press Ctrl-D or /exit to quit.".dimmed());
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
        println!("{}", format!("Ghost-OS CLI v{}", env!("CARGO_PKG_VERSION")).white());
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
        println!();
    }
}

#[derive(Debug, PartialEq, Eq)]
enum RoutedInput<'a> {
    Command(&'a str),
    Message(&'a str),
}

fn route_input(input: &str) -> RoutedInput<'_> {
    if input.starts_with("//") {
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
