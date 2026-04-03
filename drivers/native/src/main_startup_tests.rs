use super::{resolve_entry_route, route_entry, EntryRoute};

#[test]
fn resolve_entry_route_defaults_to_oneshot() {
    let args = vec!["native".to_string()];
    let route = resolve_entry_route(&args).expect("default route should resolve");
    assert_eq!(route, EntryRoute::OneShot);
}

#[test]
fn resolve_entry_route_selects_sandbox_worker() {
    let args = vec!["native".to_string(), "--sandbox-worker".to_string()];
    let route = resolve_entry_route(&args).expect("sandbox worker route should resolve");
    assert_eq!(route, EntryRoute::SandboxWorker);
}

#[test]
fn resolve_entry_route_selects_persistent() {
    let args = vec!["native".to_string(), "--persistent".to_string()];
    let route = resolve_entry_route(&args).expect("persistent route should resolve");
    assert_eq!(route, EntryRoute::Persistent);
}

#[test]
fn route_entry_surfaces_unknown_argument_error() {
    let args = vec!["native".to_string(), "--invalid".to_string()];
    let result = route_entry(&args);
    assert!(!result.handled, "unknown argument should not be handled");
    let error = result
        .error
        .expect("unknown argument should have explicit error");
    assert!(error.contains("unknown argument: --invalid"));
}

#[test]
fn route_entry_surfaces_conflicting_flag_error() {
    let args = vec![
        "native".to_string(),
        "--sandbox-worker".to_string(),
        "--persistent".to_string(),
    ];
    let result = route_entry(&args);
    assert!(!result.handled, "conflicting route should not be handled");
    let error = result
        .error
        .expect("conflicting route should have explicit error");
    assert!(error.contains("conflicting arguments"));
}
