pub(super) fn format_numbered_content(content: &str, start_line: usize) -> String {
    if content.is_empty() || start_line == 0 {
        return String::new();
    }

    let lines: Vec<&str> = content.lines().collect();
    let last_line = start_line + lines.len().saturating_sub(1);
    let width = last_line.to_string().len();

    lines
        .into_iter()
        .enumerate()
        .map(|(index, line)| format!("{:>width$} | {}", start_line + index, line, width = width))
        .collect::<Vec<_>>()
        .join("\n")
}
