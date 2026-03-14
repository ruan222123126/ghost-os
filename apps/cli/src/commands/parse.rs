use anyhow::{Result, bail};

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum CliCommand {
    Help,
    Exit,
    Clear,
    Config,
    Session,
    NewSession,
    UpdateConfig(ConfigMutation),
    LiteralHint,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum ConfigMutation {
    Model(String),
    Provider(String),
    BaseUrl(ResettableValue),
    ChatPath(ResettableValue),
    ApiKey(String),
    ClearApiKey,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum ResettableValue {
    Value(String),
    Default,
}

pub(crate) fn parse_command(input: &str) -> Result<CliCommand> {
    let mut parts = input.split_whitespace();
    let command = parts.next().unwrap_or_default();

    match command {
        "/help" | "/h" => expect_no_args(command, parts).map(|_| CliCommand::Help),
        "/exit" | "/quit" | "/q" => expect_no_args(command, parts).map(|_| CliCommand::Exit),
        "/clear" => expect_no_args(command, parts).map(|_| CliCommand::Clear),
        "/config" => expect_no_args(command, parts).map(|_| CliCommand::Config),
        "/session" => expect_no_args(command, parts).map(|_| CliCommand::Session),
        "/new-session" => expect_no_args(command, parts).map(|_| CliCommand::NewSession),
        "/model" => expect_single_arg(command, parts, "<name>")
            .map(|value| model_command(value.to_string())),
        "/provider" => expect_single_arg(command, parts, "<name>")
            .map(|value| CliCommand::UpdateConfig(ConfigMutation::Provider(value.to_string()))),
        "/base-url" => expect_single_arg(command, parts, "<url|default>").map(|value| {
            CliCommand::UpdateConfig(ConfigMutation::BaseUrl(parse_resettable_value(value)))
        }),
        "/chat-path" => expect_single_arg(command, parts, "<path|default>").map(|value| {
            CliCommand::UpdateConfig(ConfigMutation::ChatPath(parse_resettable_value(value)))
        }),
        "/api-key" => expect_single_arg(command, parts, "<value>")
            .map(|value| CliCommand::UpdateConfig(ConfigMutation::ApiKey(value.to_string()))),
        "/clear-api-key" => expect_no_args(command, parts)
            .map(|_| CliCommand::UpdateConfig(ConfigMutation::ClearApiKey)),
        "/literal" | "/l" => expect_no_args(command, parts).map(|_| CliCommand::LiteralHint),
        _ => bail!("Unknown command: {command}. Type /help for available commands."),
    }
}

fn model_command(value: String) -> CliCommand {
    CliCommand::UpdateConfig(ConfigMutation::Model(value))
}

fn parse_resettable_value(value: &str) -> ResettableValue {
    if value.eq_ignore_ascii_case("default") {
        return ResettableValue::Default;
    }
    ResettableValue::Value(value.to_string())
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

#[cfg(test)]
mod tests {
    use super::{CliCommand, ConfigMutation, ResettableValue, parse_command};

    #[test]
    fn parse_model_requires_exactly_one_arg() {
        let command = parse_command("/model gpt-4o").unwrap();
        assert_eq!(
            command,
            CliCommand::UpdateConfig(ConfigMutation::Model("gpt-4o".to_string()))
        );
        assert!(parse_command("/model").is_err());
        assert!(parse_command("/model gpt-4o extra").is_err());
    }

    #[test]
    fn parse_provider_requires_exactly_one_arg() {
        let command = parse_command("/provider openai").unwrap();
        assert_eq!(
            command,
            CliCommand::UpdateConfig(ConfigMutation::Provider("openai".to_string()))
        );
        assert!(parse_command("/provider").is_err());
        assert!(parse_command("/provider openai extra").is_err());
    }

    #[test]
    fn parse_runtime_config_commands_require_valid_args() {
        assert_eq!(
            parse_command("/base-url http://127.0.0.1:11434/v1").unwrap(),
            CliCommand::UpdateConfig(ConfigMutation::BaseUrl(ResettableValue::Value(
                "http://127.0.0.1:11434/v1".to_string()
            )))
        );
        assert_eq!(
            parse_command("/chat-path default").unwrap(),
            CliCommand::UpdateConfig(ConfigMutation::ChatPath(ResettableValue::Default))
        );
        assert_eq!(
            parse_command("/api-key secret-key").unwrap(),
            CliCommand::UpdateConfig(ConfigMutation::ApiKey("secret-key".to_string()))
        );
        assert_eq!(
            parse_command("/clear-api-key").unwrap(),
            CliCommand::UpdateConfig(ConfigMutation::ClearApiKey)
        );
        assert!(parse_command("/base-url").is_err());
        assert!(parse_command("/chat-path").is_err());
        assert!(parse_command("/api-key").is_err());
        assert!(parse_command("/clear-api-key extra").is_err());
    }

    #[test]
    fn parse_session_commands_require_no_args() {
        assert_eq!(parse_command("/session").unwrap(), CliCommand::Session);
        assert_eq!(
            parse_command("/new-session").unwrap(),
            CliCommand::NewSession
        );
        assert!(parse_command("/session extra").is_err());
        assert!(parse_command("/new-session extra").is_err());
    }

    #[test]
    fn parse_unknown_command_returns_local_error() {
        let err = parse_command("/nope").unwrap_err().to_string();
        assert!(err.contains("Unknown command: /nope"));
    }
}
