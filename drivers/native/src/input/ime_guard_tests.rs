#[cfg(target_os = "linux")]
use super::{
    first_non_empty_line, ibus_ascii_engine_rank, parse_fcitx_state_text, parse_ibus_engine_line,
    select_best_ibus_ascii_engine, stdout_trimmed,
};
#[cfg(target_os = "linux")]
use std::os::unix::process::ExitStatusExt;
#[cfg(target_os = "linux")]
use std::process::{ExitStatus, Output};

#[cfg(target_os = "linux")]
#[test]
fn ibus_ascii_engine_rank_prefers_us_layout() {
    assert!(ibus_ascii_engine_rank("xkb:us::eng") > ibus_ascii_engine_rank("xkb:fr::fra"));
    assert!(ibus_ascii_engine_rank("xkb:fr::fra") > ibus_ascii_engine_rank("pinyin"));
}

#[cfg(target_os = "linux")]
#[test]
fn parse_fcitx_state_text_handles_blank_and_invalid_values() {
    assert_eq!(parse_fcitx_state_text("2"), Some(2));
    assert_eq!(parse_fcitx_state_text("  1  "), Some(1));
    assert_eq!(parse_fcitx_state_text(""), None);
    assert_eq!(parse_fcitx_state_text("n/a"), None);
}

#[cfg(target_os = "linux")]
#[test]
fn parse_ibus_engine_line_filters_invalid_candidates() {
    assert_eq!(
        parse_ibus_engine_line(b"xkb:us::eng"),
        Some("xkb:us::eng".to_string())
    );
    assert_eq!(parse_ibus_engine_line(b""), None);
    assert_eq!(parse_ibus_engine_line(b"not-an-engine"), None);
    assert_eq!(parse_ibus_engine_line(b"xkb: us::eng"), None);
}

#[cfg(target_os = "linux")]
#[test]
fn select_best_ibus_ascii_engine_picks_highest_ranked_candidate() {
    let sample = b"pinyin\nxkb:fr::fra\nxkb:us::eng\n";
    assert_eq!(
        select_best_ibus_ascii_engine(sample),
        Some("xkb:us::eng".to_string())
    );
}

#[cfg(target_os = "linux")]
#[test]
fn first_non_empty_line_prefers_stderr_then_stdout() {
    let output = make_output(1, "stdout line\n", "\n stderr line \n");
    assert_eq!(first_non_empty_line(&output), Some("stderr line"));
}

#[cfg(target_os = "linux")]
#[test]
fn stdout_trimmed_removes_surrounding_spaces() {
    let output = make_output(0, "  hello  \n", "");
    assert_eq!(stdout_trimmed(&output), "hello");
}

#[cfg(target_os = "linux")]
fn make_output(code: i32, stdout: &str, stderr: &str) -> Output {
    Output {
        status: ExitStatus::from_raw(code),
        stdout: stdout.as_bytes().to_vec(),
        stderr: stderr.as_bytes().to_vec(),
    }
}
