use crate::client::BridgeClient;
use crate::config::Config;

use super::{
    CliCommand, CommandAction, CommandContext, CommandOutput, ConfigChangeKind, ConfigMutation,
    ResettableValue, SessionStatus, build_config_update, execute_command,
};

#[test]
fn execute_command_clears_session_without_touching_client() {
    let client = dummy_client();
    let mut current_session_id = Some("session-123".to_string());

    let result = execute_command(
        CliCommand::NewSession,
        CommandContext::new(&client, &mut current_session_id),
    )
    .unwrap();

    assert_eq!(result.action, CommandAction::Continue);
    assert_eq!(result.output, CommandOutput::SessionCleared);
    assert!(current_session_id.is_none());
}

#[test]
fn execute_command_reports_active_session() {
    let client = dummy_client();
    let mut current_session_id = Some("session-123".to_string());

    let result = execute_command(
        CliCommand::Session,
        CommandContext::new(&client, &mut current_session_id),
    )
    .unwrap();

    assert_eq!(result.action, CommandAction::Continue);
    assert_eq!(
        result.output,
        CommandOutput::SessionStatus(SessionStatus::Active("session-123".to_string()))
    );
}

#[test]
fn execute_command_returns_exit_action_for_exit_command() {
    let client = dummy_client();
    let mut current_session_id = None;

    let result = execute_command(
        CliCommand::Exit,
        CommandContext::new(&client, &mut current_session_id),
    )
    .unwrap();

    assert_eq!(result.action, CommandAction::Exit);
    assert_eq!(result.output, CommandOutput::None);
}

#[test]
fn build_config_update_maps_default_resets_to_empty_strings() {
    let (base_url, base_kind) =
        build_config_update(ConfigMutation::BaseUrl(ResettableValue::Default));
    let (chat_path, chat_kind) =
        build_config_update(ConfigMutation::ChatPath(ResettableValue::Default));

    assert_eq!(base_url.base_url.as_deref(), Some(""));
    assert_eq!(chat_path.chat_path.as_deref(), Some(""));
    assert_eq!(base_kind, ConfigChangeKind::BaseUrlReset);
    assert_eq!(chat_kind, ConfigChangeKind::ChatPathReset);
}

#[test]
fn build_config_update_maps_api_key_and_model_updates() {
    let (api_key, api_key_kind) = build_config_update(ConfigMutation::ApiKey("secret".to_string()));
    let (model, model_kind) = build_config_update(ConfigMutation::Model("gpt-4o".to_string()));

    assert_eq!(api_key.api_key.as_deref(), Some("secret"));
    assert_eq!(model.model.as_deref(), Some("gpt-4o"));
    assert_eq!(api_key_kind, ConfigChangeKind::ApiKeyUpdated);
    assert_eq!(model_kind, ConfigChangeKind::Model);
}

fn dummy_client() -> BridgeClient {
    BridgeClient::new(Config {
        bridge_url: "http://127.0.0.1:18080".to_string(),
        timeout_secs: 1,
        startup_project_root: "/tmp/ghost-os".to_string(),
    })
    .unwrap()
}
