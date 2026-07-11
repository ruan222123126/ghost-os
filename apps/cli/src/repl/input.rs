#[derive(Debug, PartialEq, Eq)]
pub(super) enum RoutedInput<'a> {
    Command(&'a str),
    Message(&'a str),
}

pub(super) fn route_input(input: &str) -> RoutedInput<'_> {
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
