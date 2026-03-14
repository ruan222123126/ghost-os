mod ask_human;
mod input;
mod view;

use anyhow::Result;
use rustyline::DefaultEditor;
use rustyline::error::ReadlineError;

use crate::client::BridgeClient;
use crate::commands::{
    CommandAction, CommandContext, execute_command, parse_command, render_output,
};
use crate::types::AgentPayload;

use input::{RoutedInput, route_input};

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
        view::print_welcome(&self.client);
        let prompt = view::main_prompt();

        loop {
            match self.read_main_line(&prompt)? {
                LoopEvent::Continue => continue,
                LoopEvent::Exit => break,
                LoopEvent::Line(line) => match self.handle_input_line(&line) {
                    Ok(RunAction::Continue) => {}
                    Ok(RunAction::Exit) => break,
                    Err(err) => view::print_error(&err.to_string()),
                },
            }
        }

        Ok(())
    }

    fn read_main_line(&mut self, prompt: &str) -> Result<LoopEvent> {
        match self.editor.readline(prompt) {
            Ok(line) => Ok(LoopEvent::Line(line)),
            Err(ReadlineError::Interrupted) => {
                view::print_input_cancelled();
                Ok(LoopEvent::Continue)
            }
            Err(ReadlineError::Eof) => {
                view::print_goodbye();
                Ok(LoopEvent::Exit)
            }
            Err(err) => Err(err.into()),
        }
    }

    fn handle_input_line(&mut self, line: &str) -> Result<RunAction> {
        let input = line.trim();
        if input.is_empty() {
            return Ok(RunAction::Continue);
        }
        let _ = self.editor.add_history_entry(input);

        match route_input(input) {
            RoutedInput::Command(command) => self.handle_command_input(command),
            RoutedInput::Message(message) => self.handle_message_input(message),
        }
    }

    fn handle_command_input(&mut self, input: &str) -> Result<RunAction> {
        let command = parse_command(input)?;
        let execution = execute_command(
            command,
            CommandContext::new(&self.client, &mut self.current_session_id),
        )?;
        render_output(&execution.output)?;

        if execution.action == CommandAction::Exit {
            view::print_goodbye();
            return Ok(RunAction::Exit);
        }
        Ok(RunAction::Continue)
    }

    fn handle_message_input(&mut self, message: &str) -> Result<RunAction> {
        view::print_thinking();
        let payload = self
            .client
            .send_message(message, self.current_session_id.as_deref())?;
        self.drain_agent_payload(payload)?;
        Ok(RunAction::Continue)
    }

    fn drain_agent_payload(&mut self, payload: AgentPayload) -> Result<()> {
        let mut current = payload;

        loop {
            self.remember_session_id(current.session_id());
            if let Some(question) = current.as_awaiting_human() {
                current =
                    ask_human::continue_awaiting_human(&self.client, &mut self.editor, question)?;
                continue;
            }
            if let Some(reply) = current.into_success() {
                view::print_reply(&reply.message);
            }
            return Ok(());
        }
    }

    fn remember_session_id(&mut self, session_id: &str) {
        let session_id = session_id.trim();
        if session_id.is_empty() {
            return;
        }
        self.current_session_id = Some(session_id.to_string());
    }
}

enum LoopEvent {
    Continue,
    Exit,
    Line(String),
}

enum RunAction {
    Continue,
    Exit,
}
