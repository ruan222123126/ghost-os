use std::collections::HashMap;

use super::command::CodexCommand;

pub(crate) struct CodexCommandManager {
    next_id: u64,
    commands: HashMap<String, CodexCommand>,
}

impl CodexCommandManager {
    pub(crate) fn new() -> Self {
        Self {
            next_id: 0,
            commands: HashMap::new(),
        }
    }

    pub(super) fn next_command_id(&mut self) -> (u64, String) {
        self.next_id += 1;
        let command_id = format!("codex-cli-{}", self.next_id);
        (self.next_id, command_id)
    }

    pub(super) fn insert(&mut self, command: CodexCommand) {
        self.commands.insert(command.id().to_string(), command);
    }

    pub(super) fn get_mut(&mut self, id: &str) -> Option<&mut CodexCommand> {
        self.commands.get_mut(id)
    }

    pub(super) fn contains(&self, id: &str) -> bool {
        self.commands.contains_key(id)
    }

    pub(super) fn find_by_session_id(&mut self, session_id: &str) -> Option<&mut CodexCommand> {
        let command_id = latest_matching_command_id(&self.commands, session_id)?;
        self.commands.get_mut(&command_id)
    }
}

fn latest_matching_command_id(
    commands: &HashMap<String, CodexCommand>,
    session_id: &str,
) -> Option<String> {
    let mut best_match: Option<(&str, u64)> = None;
    for (id, command) in commands {
        if command.session_id_value().as_deref() != Some(session_id) {
            continue;
        }
        if best_match.is_none_or(|(_, seq)| command.seq() > seq) {
            best_match = Some((id.as_str(), command.seq()));
        }
    }
    best_match.map(|(id, _)| id.to_string())
}
