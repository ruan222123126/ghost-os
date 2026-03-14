use std::env;
use std::io::{self, Write};
use std::process::Command;

use anyhow::{Result, bail};
use colored::Colorize;

use crate::client::BridgeClient;
use crate::types::ConfigUpdate;

pub enum CommandAction {
    Continue,
    Exit,
}

#[derive(Debug)]
enum ParsedCommand<'a> {
    Help,
    Exit,
    Clear,
    Config,
    Session,
    NewSession,
    Model(&'a str),
    Provider(&'a str),
    BaseURL(&'a str),
    ChatPath(&'a str),
    APIKey(&'a str),
    ClearAPIKey,
    LiteralHint,
}

pub fn handle_command(
    input: &str,
    client: &BridgeClient,
    current_session_id: &mut Option<String>,
) -> Result<CommandAction> {
    match parse_command(input)? {
        ParsedCommand::Help => {
            print_help();
            Ok(CommandAction::Continue)
        }
        ParsedCommand::Exit => Ok(CommandAction::Exit),
        ParsedCommand::Clear => {
            clear_screen()?;
            Ok(CommandAction::Continue)
        }
        ParsedCommand::Config => {
            let cfg = client.get_config()?;
            println!("{}", "Bridge Config".bright_black());
            println!("provider  : {}", cfg.provider.white());
            println!(
                "type      : {}",
                provider_type_label(&cfg.provider_type).white()
            );
            println!("model     : {}", cfg.model.white());
            println!("base_url  : {}", cfg.base_url.white());
            if cfg.chat_path.trim().is_empty() {
                println!("chat_path : {}", "(default)".dimmed());
            } else {
                println!("chat_path : {}", cfg.chat_path.white());
            }
            println!(
                "api_key   : {}",
                if cfg.api_key_set {
                    "set".green()
                } else {
                    "not set".yellow()
                }
            );
            Ok(CommandAction::Continue)
        }
        ParsedCommand::Session => {
            match current_session_id.as_deref() {
                Some(session_id) if !session_id.trim().is_empty() => {
                    println!("Current session: {}", session_id.bright_cyan());
                }
                _ => println!("{}", "No active session".dimmed()),
            }
            Ok(CommandAction::Continue)
        }
        ParsedCommand::NewSession => {
            *current_session_id = None;
            println!(
                "{}",
                "Session cleared. Next message will start a new conversation.".dimmed()
            );
            Ok(CommandAction::Continue)
        }
        ParsedCommand::Model(model) => {
            let update = ConfigUpdate {
                model: Some(model.to_string()),
                ..ConfigUpdate::default()
            };
            let cfg = client.update_config(&update)?;
            println!("{}", format!("Model changed to: {}", cfg.model).green());
            Ok(CommandAction::Continue)
        }
        ParsedCommand::Provider(provider) => {
            let update = ConfigUpdate {
                provider: Some(provider.to_string()),
                ..ConfigUpdate::default()
            };
            let cfg = client.update_config(&update)?;
            println!(
                "{}",
                format!("Provider changed to: {}", cfg.provider).green()
            );
            Ok(CommandAction::Continue)
        }
        ParsedCommand::BaseURL(base_url) => {
            let reset = base_url.eq_ignore_ascii_case("default");
            let update = ConfigUpdate {
                base_url: Some(if reset {
                    String::new()
                } else {
                    base_url.to_string()
                }),
                ..ConfigUpdate::default()
            };
            let cfg = client.update_config(&update)?;
            if reset {
                println!("{}", format!("Base URL reset to: {}", cfg.base_url).green());
            } else {
                println!(
                    "{}",
                    format!("Base URL changed to: {}", cfg.base_url).green()
                );
            }
            Ok(CommandAction::Continue)
        }
        ParsedCommand::ChatPath(chat_path) => {
            let reset = chat_path.eq_ignore_ascii_case("default");
            let update = ConfigUpdate {
                chat_path: Some(if reset {
                    String::new()
                } else {
                    chat_path.to_string()
                }),
                ..ConfigUpdate::default()
            };
            let cfg = client.update_config(&update)?;
            if cfg.chat_path.trim().is_empty() {
                println!("{}", "Chat path reset to provider default".green());
            } else {
                println!(
                    "{}",
                    format!("Chat path changed to: {}", cfg.chat_path).green()
                );
            }
            Ok(CommandAction::Continue)
        }
        ParsedCommand::APIKey(api_key) => {
            let update = ConfigUpdate {
                api_key: Some(api_key.to_string()),
                ..ConfigUpdate::default()
            };
            client.update_config(&update)?;
            println!("{}", "API key updated".green());
            Ok(CommandAction::Continue)
        }
        ParsedCommand::ClearAPIKey => {
            let update = ConfigUpdate {
                api_key: Some(String::new()),
                ..ConfigUpdate::default()
            };
            client.update_config(&update)?;
            println!("{}", "API key cleared".green());
            Ok(CommandAction::Continue)
        }
        ParsedCommand::LiteralHint => {
            println!(
                "{}",
                "Use //text to send a literal /text message to the model.".dimmed()
            );
            Ok(CommandAction::Continue)
        }
    }
}

fn parse_command(input: &str) -> Result<ParsedCommand<'_>> {
    let mut parts = input.split_whitespace();
    let command = parts.next().unwrap_or_default();

    match command {
        "/help" | "/h" => expect_no_args(command, parts).map(|_| ParsedCommand::Help),
        "/exit" | "/quit" | "/q" => expect_no_args(command, parts).map(|_| ParsedCommand::Exit),
        "/clear" => expect_no_args(command, parts).map(|_| ParsedCommand::Clear),
        "/config" => expect_no_args(command, parts).map(|_| ParsedCommand::Config),
        "/session" => expect_no_args(command, parts).map(|_| ParsedCommand::Session),
        "/new-session" => expect_no_args(command, parts).map(|_| ParsedCommand::NewSession),
        "/model" => expect_single_arg(command, parts, "<name>").map(ParsedCommand::Model),
        "/provider" => expect_single_arg(command, parts, "<name>").map(ParsedCommand::Provider),
        "/base-url" => {
            expect_single_arg(command, parts, "<url|default>").map(ParsedCommand::BaseURL)
        }
        "/chat-path" => {
            expect_single_arg(command, parts, "<path|default>").map(ParsedCommand::ChatPath)
        }
        "/api-key" => expect_single_arg(command, parts, "<value>").map(ParsedCommand::APIKey),
        "/clear-api-key" => expect_no_args(command, parts).map(|_| ParsedCommand::ClearAPIKey),
        "/literal" | "/l" => expect_no_args(command, parts).map(|_| ParsedCommand::LiteralHint),
        _ => bail!("Unknown command: {command}. Type /help for available commands."),
    }
}

fn expect_no_args<'a>(command: &str, mut parts: impl Iterator<Item = &'a str>) -> Result<()> {
    if parts.next().is_some() {
        return usage_error(command, "");
    }
    Ok(())
}

fn expect_single_arg<'a>(
    command: &str,
    mut parts: impl Iterator<Item = &'a str>,
    usage_suffix: &str,
) -> Result<&'a str> {
    let Some(value) = parts.next() else {
        return usage_error(command, usage_suffix);
    };
    if parts.next().is_some() {
        return usage_error(command, usage_suffix);
    }
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return usage_error(command, usage_suffix);
    }
    Ok(trimmed)
}

fn usage_error<T>(command: &str, usage_suffix: &str) -> Result<T> {
    if usage_suffix.is_empty() {
        bail!("Usage: {command}");
    }
    bail!("Usage: {command} {usage_suffix}");
}

fn clear_screen() -> Result<()> {
    let mut stdout = io::stdout();
    clear_screen_with(prefer_ansi_clear(), run_system_clear, &mut stdout)
}

fn clear_screen_with(
    prefer_ansi: bool,
    run_system_clear: impl FnOnce() -> io::Result<bool>,
    out: &mut dyn Write,
) -> Result<()> {
    if prefer_ansi {
        write_ansi_clear(out)?;
        return Ok(());
    }

    match run_system_clear() {
        Ok(true) => Ok(()),
        _ => {
            write_ansi_clear(out)?;
            Ok(())
        }
    }
}

fn run_system_clear() -> io::Result<bool> {
    let status = if cfg!(windows) {
        Command::new("cmd").args(["/C", "cls"]).status()
    } else {
        Command::new("clear").status()
    };
    status.map(|value| value.success())
}

fn write_ansi_clear(out: &mut dyn Write) -> io::Result<()> {
    // ANSI: clear screen + move cursor to top-left.
    write!(out, "\x1B[2J\x1B[H")?;
    out.flush()
}

fn prefer_ansi_clear() -> bool {
    if cfg!(windows) {
        return env::var_os("WT_SESSION").is_some()
            || env::var_os("ANSICON").is_some()
            || env::var("TERM").map(|term| term != "dumb").unwrap_or(false);
    }
    env::var("TERM").map(|term| term != "dumb").unwrap_or(true)
}

fn provider_type_label(provider_type: &str) -> &str {
    match provider_type.trim().to_ascii_lowercase().as_str() {
        "openai" => "OpenAI",
        "codex" => "Codex",
        "anthropic" => "Anthropic",
        "custom" => "OpenAI-Compatible",
        _ => provider_type,
    }
}

fn print_help() {
    println!("{}", "Commands".bright_black());
    println!("/help, /h                Show help");
    println!("/config                  Show current bridge configuration");
    println!("/session                 Show current conversation session id");
    println!("/new-session             Start a fresh conversation session");
    println!("/model <name>            Switch model");
    println!("/provider <name>         Switch active provider by saved name");
    println!("/base-url <url|default>  Change provider base URL or reset default");
    println!("/chat-path <path|default> Change provider chat path or reset default");
    println!("/api-key <value>         Update runtime API key");
    println!("/clear-api-key           Clear runtime API key");
    println!("/clear                   Clear terminal");
    println!("/literal, /l             Show slash-literal usage");
    println!("/exit, /quit, /q         Exit CLI");
    println!();
    println!(
        "{}",
        "Tip: use //text to send /text as a normal message.".dimmed()
    );
}

#[cfg(test)]
mod tests {
    use std::cell::Cell;
    use std::io;

    use super::{clear_screen_with, parse_command};

    #[test]
    fn parse_model_requires_exactly_one_arg() {
        assert!(parse_command("/model gpt-4o").is_ok());
        assert!(parse_command("/model").is_err());
        assert!(parse_command("/model gpt-4o extra").is_err());
    }

    #[test]
    fn parse_provider_requires_exactly_one_arg() {
        assert!(parse_command("/provider openai").is_ok());
        assert!(parse_command("/provider").is_err());
        assert!(parse_command("/provider openai extra").is_err());
    }

    #[test]
    fn parse_runtime_config_commands_require_valid_args() {
        assert!(parse_command("/base-url http://127.0.0.1:11434/v1").is_ok());
        assert!(parse_command("/chat-path /v1/messages").is_ok());
        assert!(parse_command("/api-key secret-key").is_ok());
        assert!(parse_command("/clear-api-key").is_ok());
        assert!(parse_command("/base-url").is_err());
        assert!(parse_command("/chat-path").is_err());
        assert!(parse_command("/api-key").is_err());
        assert!(parse_command("/clear-api-key extra").is_err());
    }

    #[test]
    fn parse_session_commands_require_no_args() {
        assert!(parse_command("/session").is_ok());
        assert!(parse_command("/new-session").is_ok());
        assert!(parse_command("/session extra").is_err());
        assert!(parse_command("/new-session extra").is_err());
    }

    #[test]
    fn parse_unknown_command_returns_local_error() {
        let err = parse_command("/nope").unwrap_err().to_string();
        assert!(err.contains("Unknown command: /nope"));
    }

    #[test]
    fn clear_screen_falls_back_to_ansi_when_system_clear_fails() {
        let called = Cell::new(false);
        let mut buf = Vec::new();

        clear_screen_with(
            false,
            || {
                called.set(true);
                Ok(false)
            },
            &mut buf,
        )
        .unwrap();

        assert!(called.get());
        assert_eq!(buf, b"\x1B[2J\x1B[H");
    }

    #[test]
    fn clear_screen_uses_system_clear_when_ansi_not_preferred() {
        let called = Cell::new(false);
        let mut buf = Vec::new();

        clear_screen_with(
            false,
            || {
                called.set(true);
                Ok(true)
            },
            &mut buf,
        )
        .unwrap();

        assert!(called.get());
        assert!(buf.is_empty());
    }

    #[test]
    fn clear_screen_uses_ansi_first_when_preferred() {
        let called = Cell::new(false);
        let mut buf = Vec::new();

        clear_screen_with(
            true,
            || {
                called.set(true);
                Ok(true)
            },
            &mut buf,
        )
        .unwrap();

        assert!(!called.get());
        assert_eq!(buf, b"\x1B[2J\x1B[H");
    }

    #[test]
    fn clear_screen_falls_back_to_ansi_when_system_clear_errors() {
        let called = Cell::new(false);
        let mut buf = Vec::new();

        clear_screen_with(
            false,
            || {
                called.set(true);
                Err(io::Error::other("failed"))
            },
            &mut buf,
        )
        .unwrap();

        assert!(called.get());
        assert_eq!(buf, b"\x1B[2J\x1B[H");
    }
}
