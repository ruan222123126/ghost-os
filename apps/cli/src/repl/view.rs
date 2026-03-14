use colored::Colorize;

use crate::client::BridgeClient;
use crate::types::AskHumanOption;

pub(super) fn print_welcome(client: &BridgeClient) {
    println!(
        "{}",
        format!("Ghost-OS CLI v{}", env!("CARGO_PKG_VERSION")).white()
    );
    println!(
        "{} {}",
        "Connected to bridge at".dimmed(),
        client.bridge_url().white()
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

pub(super) fn main_prompt() -> String {
    "ghost> ".bright_black().to_string()
}

pub(super) fn answer_prompt() -> String {
    "Your answer (/cancel)> ".bright_yellow().to_string()
}

pub(super) fn choice_prompt(multiple: bool) -> String {
    if multiple {
        return "Choice(s)> ".bright_yellow().to_string();
    }
    "Choice> ".bright_yellow().to_string()
}

pub(super) fn custom_prompt() -> String {
    "Custom answer (/cancel)> ".bright_yellow().to_string()
}

pub(super) fn print_input_cancelled() {
    println!(
        "{}",
        "Input cancelled. Press Ctrl-D or /exit to quit.".dimmed()
    );
}

pub(super) fn print_goodbye() {
    println!("{}", "Goodbye.".dimmed());
}

pub(super) fn print_thinking() {
    println!("{}", "[Agent is thinking...]".bright_black());
}

pub(super) fn print_reply(message: &str) {
    println!();
    println!("{}", message.white());
    println!();
}

pub(super) fn print_error(message: &str) {
    println!();
    println!("{}", format!("Error: {message}").red());
    println!();
}

pub(super) fn print_pending_question(prompt: &str) {
    println!();
    println!("{}", "Agent is asking for your input:".yellow());
    println!("{}", prompt.white());
    println!();
}

pub(super) fn print_pending_options(options: &[AskHumanOption], multiple: bool) {
    println!("{}", "Options:".bright_yellow());
    for (index, option) in options.iter().enumerate() {
        let suffix = option_suffix(option);
        println!(
            "{} {}{}",
            format!("{:>2}.", index + 1).bright_black(),
            option.label.white(),
            suffix.bright_black(),
        );
    }
    println!("{}", selection_help(multiple).dimmed());
}

fn option_suffix(option: &AskHumanOption) -> &'static str {
    if option.allow_custom.unwrap_or(false) {
        return " (custom)";
    }
    ""
}

fn selection_help(multiple: bool) -> &'static str {
    if multiple {
        return "Select one or more numbers separated by commas, or /cancel.";
    }
    "Select one number, or /cancel."
}
