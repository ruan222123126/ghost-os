use std::io::{self, Write};

use colored::Colorize;

use crate::types::BridgeConfig;

use super::execute::{CommandOutput, ConfigChangeKind, ConfigUpdateFeedback, SessionStatus};

#[derive(Debug, Clone, PartialEq, Eq)]
struct RenderedLine {
    text: String,
    style: LineStyle,
}

impl RenderedLine {
    fn new(text: impl Into<String>, style: LineStyle) -> Self {
        Self {
            text: text.into(),
            style,
        }
    }

    fn blank() -> Self {
        Self::new("", LineStyle::Plain)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum LineStyle {
    Plain,
    BrightBlack,
    White,
    Green,
    Yellow,
    BrightCyan,
}

pub(crate) fn render_output(output: &CommandOutput) -> io::Result<()> {
    let lines = describe_output(output);
    let mut stdout = io::stdout();
    write_lines(&lines, &mut stdout)
}

fn write_lines(lines: &[RenderedLine], out: &mut dyn Write) -> io::Result<()> {
    for line in lines {
        write_line(line, out)?;
    }
    Ok(())
}

fn write_line(line: &RenderedLine, out: &mut dyn Write) -> io::Result<()> {
    if line.text.is_empty() {
        return writeln!(out);
    }
    writeln!(out, "{}", style_text(line))
}

fn style_text(line: &RenderedLine) -> String {
    match line.style {
        LineStyle::Plain => line.text.clone(),
        LineStyle::BrightBlack => line.text.bright_black().to_string(),
        LineStyle::White => line.text.white().to_string(),
        LineStyle::Green => line.text.green().to_string(),
        LineStyle::Yellow => line.text.yellow().to_string(),
        LineStyle::BrightCyan => line.text.bright_cyan().to_string(),
    }
}

fn describe_output(output: &CommandOutput) -> Vec<RenderedLine> {
    match output {
        CommandOutput::None => Vec::new(),
        CommandOutput::Help => help_lines(),
        CommandOutput::ConfigSnapshot(config) => config_lines(config),
        CommandOutput::SessionStatus(status) => session_lines(status),
        CommandOutput::SessionCleared => vec![dimmed_line(
            "Session cleared. Next message will start a new conversation.",
        )],
        CommandOutput::ConfigUpdated(feedback) => config_update_lines(feedback),
        CommandOutput::LiteralHint => {
            vec![dimmed_line(
                "Use //text to send a literal /text message to the model.",
            )]
        }
    }
}

fn help_lines() -> Vec<RenderedLine> {
    let commands = [
        "/help, /h                Show help",
        "/config                  Show current bridge configuration",
        "/session                 Show current conversation session id",
        "/new-session             Start a fresh conversation session",
        "/model <name>            Switch model",
        "/provider <name>         Switch active provider by saved name",
        "/base-url <url|default>  Change provider base URL or reset default",
        "/chat-path <path|default> Change provider chat path or reset default",
        "/api-key <value>         Update runtime API key",
        "/clear-api-key           Clear runtime API key",
        "/clear                   Clear terminal",
        "/literal, /l             Show slash-literal usage",
        "/exit, /quit, /q         Exit CLI",
    ];

    let mut lines = vec![RenderedLine::new("Commands", LineStyle::BrightBlack)];
    lines.extend(commands.into_iter().map(plain_line));
    lines.push(RenderedLine::blank());
    lines.push(dimmed_line(
        "Tip: use //text to send /text as a normal message.",
    ));
    lines
}

fn config_lines(config: &BridgeConfig) -> Vec<RenderedLine> {
    let chat_path = if config.chat_path.trim().is_empty() {
        "(default)"
    } else {
        config.chat_path.as_str()
    };
    let api_key = if config.api_key_set { "set" } else { "not set" };
    let provider_type = provider_type_label(&config.provider_type);

    vec![
        RenderedLine::new("Bridge Config", LineStyle::BrightBlack),
        white_line(format!("provider  : {}", config.provider)),
        white_line(format!("type      : {provider_type}")),
        white_line(format!("model     : {}", config.model)),
        white_line(format!("base_url  : {}", config.base_url)),
        white_line(format!("chat_path : {chat_path}")),
        RenderedLine::new(
            format!("api_key   : {api_key}"),
            api_key_style(config.api_key_set),
        ),
    ]
}

fn session_lines(status: &SessionStatus) -> Vec<RenderedLine> {
    match status {
        SessionStatus::Active(session_id) => {
            vec![RenderedLine::new(
                format!("Current session: {session_id}"),
                LineStyle::BrightCyan,
            )]
        }
        SessionStatus::Inactive => vec![dimmed_line("No active session")],
    }
}

fn config_update_lines(feedback: &ConfigUpdateFeedback) -> Vec<RenderedLine> {
    let message = match feedback.kind {
        ConfigChangeKind::Model => format!("Model changed to: {}", feedback.config.model),
        ConfigChangeKind::Provider => {
            format!("Provider changed to: {}", feedback.config.provider)
        }
        ConfigChangeKind::BaseUrlSet => {
            format!("Base URL changed to: {}", feedback.config.base_url)
        }
        ConfigChangeKind::BaseUrlReset => {
            format!("Base URL reset to: {}", feedback.config.base_url)
        }
        ConfigChangeKind::ChatPathSet => {
            format!("Chat path changed to: {}", feedback.config.chat_path)
        }
        ConfigChangeKind::ChatPathReset => "Chat path reset to provider default".to_string(),
        ConfigChangeKind::ApiKeyUpdated => "API key updated".to_string(),
        ConfigChangeKind::ApiKeyCleared => "API key cleared".to_string(),
    };
    vec![RenderedLine::new(message, LineStyle::Green)]
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

fn api_key_style(api_key_set: bool) -> LineStyle {
    if api_key_set {
        LineStyle::Green
    } else {
        LineStyle::Yellow
    }
}

fn plain_line(text: &str) -> RenderedLine {
    RenderedLine::new(text, LineStyle::Plain)
}

fn white_line(text: impl Into<String>) -> RenderedLine {
    RenderedLine::new(text, LineStyle::White)
}

fn dimmed_line(text: &str) -> RenderedLine {
    RenderedLine::new(text, LineStyle::BrightBlack)
}

#[cfg(test)]
mod tests {
    use super::{
        BridgeConfig, CommandOutput, ConfigChangeKind, ConfigUpdateFeedback, LineStyle,
        SessionStatus, describe_output,
    };

    #[test]
    fn describe_output_formats_config_snapshot() {
        let config = BridgeConfig {
            provider: "local".to_string(),
            provider_type: "custom".to_string(),
            base_url: "http://127.0.0.1:11434/v1".to_string(),
            model: "gpt-4o".to_string(),
            chat_path: String::new(),
            project_root: String::new(),
            max_turns: 20,
            llm_completion_retry_count: 1,
            llm_completion_retry_interval_ms: 200,
            api_key_set: false,
            model_selection_enabled: false,
            session_human_log_full_enabled: false,
            session_system_prompt_visible_enabled: true,
            assistant_markdown_enabled: true,
            tool_call_compact_output_enabled: false,
            memory_mode_enabled: false,
            microcompact_enabled: false,
            web_search_tavily_url: String::new(),
            web_search_exa_url: String::new(),
            web_search_tavily_api_key_set: false,
            web_search_exa_api_key_set: false,
        };

        let lines = describe_output(&CommandOutput::ConfigSnapshot(config));

        assert_eq!(lines[0].text, "Bridge Config");
        assert_eq!(lines[0].style, LineStyle::BrightBlack);
        assert_eq!(lines[2].text, "type      : OpenAI-Compatible");
        assert_eq!(lines[5].text, "chat_path : (default)");
        assert_eq!(lines[6].text, "api_key   : not set");
        assert_eq!(lines[6].style, LineStyle::Yellow);
    }

    #[test]
    fn describe_output_formats_config_update_feedback() {
        let feedback = ConfigUpdateFeedback {
            kind: ConfigChangeKind::ChatPathReset,
            config: BridgeConfig {
                provider: "openai".to_string(),
                provider_type: "openai".to_string(),
                base_url: "https://api.openai.com/v1".to_string(),
                model: "gpt-4o".to_string(),
                chat_path: String::new(),
                project_root: String::new(),
                max_turns: 20,
                llm_completion_retry_count: 1,
                llm_completion_retry_interval_ms: 200,
                api_key_set: true,
                model_selection_enabled: true,
                session_human_log_full_enabled: false,
                session_system_prompt_visible_enabled: true,
                assistant_markdown_enabled: true,
                tool_call_compact_output_enabled: false,
                memory_mode_enabled: false,
                microcompact_enabled: false,
                web_search_tavily_url: String::new(),
                web_search_exa_url: String::new(),
                web_search_tavily_api_key_set: false,
                web_search_exa_api_key_set: false,
            },
        };

        let lines = describe_output(&CommandOutput::ConfigUpdated(feedback));

        assert_eq!(lines.len(), 1);
        assert_eq!(lines[0].text, "Chat path reset to provider default");
        assert_eq!(lines[0].style, LineStyle::Green);
    }

    #[test]
    fn describe_output_formats_session_status() {
        let active = describe_output(&CommandOutput::SessionStatus(SessionStatus::Active(
            "session-123".to_string(),
        )));
        let inactive = describe_output(&CommandOutput::SessionStatus(SessionStatus::Inactive));

        assert_eq!(active[0].text, "Current session: session-123");
        assert_eq!(active[0].style, LineStyle::BrightCyan);
        assert_eq!(inactive[0].text, "No active session");
        assert_eq!(inactive[0].style, LineStyle::BrightBlack);
    }
}
