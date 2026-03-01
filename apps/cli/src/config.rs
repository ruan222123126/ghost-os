// CLI-side configuration loading, defaults, and environment overrides.

use std::env;

use anyhow::{Context, Result, bail};
use reqwest::Url;

const DEFAULT_BRIDGE_URL: &str = "http://localhost:8080";
const DEFAULT_TIMEOUT_SECS: u64 = 30;

#[derive(Debug, Clone)]
pub struct Config {
    pub bridge_url: String,
    pub timeout_secs: u64,
}

impl Config {
    pub fn load(bridge_url_arg: Option<String>, timeout_arg: Option<u64>) -> Result<Self> {
        let bridge_url = match bridge_url_arg {
            Some(value) => validate_bridge_url(&value)?,
            None => match env::var("GHOST_BRIDGE_URL") {
                Ok(value) => validate_bridge_url(&value)?,
                Err(_) => DEFAULT_BRIDGE_URL.to_string(),
            },
        };

        let timeout_secs = resolve_timeout(timeout_arg)?;

        Ok(Self {
            bridge_url,
            timeout_secs,
        })
    }
}

fn resolve_timeout(timeout_arg: Option<u64>) -> Result<u64> {
    if let Some(value) = timeout_arg {
        return validate_timeout(value, "CLI --timeout");
    }

    match env::var("GHOST_CLI_TIMEOUT") {
        Ok(raw) => {
            let parsed = raw.trim().parse::<u64>().with_context(|| {
                format!("invalid GHOST_CLI_TIMEOUT value {raw:?}, expected integer seconds")
            })?;
            validate_timeout(parsed, "GHOST_CLI_TIMEOUT")
        }
        Err(_) => Ok(DEFAULT_TIMEOUT_SECS),
    }
}

fn validate_timeout(timeout_secs: u64, source: &str) -> Result<u64> {
    if timeout_secs == 0 {
        bail!("{source} must be greater than zero");
    }
    Ok(timeout_secs)
}

fn validate_bridge_url(raw: &str) -> Result<String> {
    let candidate = raw.trim();
    if candidate.is_empty() {
        bail!("bridge URL cannot be empty");
    }

    let parsed = Url::parse(candidate)
        .with_context(|| format!("invalid bridge URL {candidate:?}, expected http://host:port"))?;

    match parsed.scheme() {
        "http" | "https" => {}
        _ => bail!("bridge URL must use http or https"),
    }

    if parsed.host_str().is_none() {
        bail!("bridge URL must include a host");
    }

    Ok(candidate.trim_end_matches('/').to_string())
}

#[cfg(test)]
mod tests {
    use super::{Config, validate_bridge_url, validate_timeout};

    #[test]
    fn bridge_url_validation_accepts_http_and_trims_trailing_slash() {
        let value = validate_bridge_url("http://localhost:8080/").unwrap();
        assert_eq!(value, "http://localhost:8080");
    }

    #[test]
    fn bridge_url_validation_rejects_non_http_scheme() {
        assert!(validate_bridge_url("ftp://localhost").is_err());
    }

    #[test]
    fn timeout_must_be_positive() {
        assert!(validate_timeout(0, "test").is_err());
        assert_eq!(validate_timeout(30, "test").unwrap(), 30);
    }

    #[test]
    fn config_load_uses_cli_args_when_provided() {
        let cfg = Config::load(Some("http://127.0.0.1:18080".to_string()), Some(12)).unwrap();
        assert_eq!(cfg.bridge_url, "http://127.0.0.1:18080");
        assert_eq!(cfg.timeout_secs, 12);
    }
}
