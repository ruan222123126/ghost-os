// Web fetch safety guards (URL validation + SSRF checks) and HTML-to-text conversion.

use reqwest::Url;
use reqwest::blocking::Client;
use std::io::Read;
use std::net::{IpAddr, ToSocketAddrs};
use std::time::Duration;

pub(crate) fn validate_https_url(raw_url: &str) -> Result<Url, String> {
    let trimmed = raw_url.trim();
    if trimmed.is_empty() {
        return Err("url is required".to_string());
    }

    let parsed = Url::parse(trimmed).map_err(|err| format!("invalid URL: {err}"))?;
    if parsed.scheme() != "https" {
        return Err("only HTTPS URLs allowed".to_string());
    }

    let host = parsed
        .host_str()
        .ok_or_else(|| "URL must include host".to_string())?;
    if host.eq_ignore_ascii_case("localhost") || host.ends_with(".localhost") {
        return Err("private IP addresses blocked".to_string());
    }

    if let Ok(ip) = host.parse::<IpAddr>() {
        if is_private_ip(ip) {
            return Err("private IP addresses blocked".to_string());
        }
    } else {
        let port = parsed.port_or_known_default().unwrap_or(443);
        let mut resolved_any = false;
        for addr in (host, port)
            .to_socket_addrs()
            .map_err(|err| format!("host resolution failed: {err}"))?
        {
            resolved_any = true;
            if is_private_ip(addr.ip()) {
                return Err("private IP addresses blocked".to_string());
            }
        }

        if !resolved_any {
            return Err("host resolution failed".to_string());
        }
    }

    Ok(parsed)
}

pub(crate) fn fetch_and_convert_webpage(url: &Url, max_bytes: usize) -> Result<String, String> {
    let client = Client::builder()
        .timeout(Duration::from_secs(10))
        .user_agent("Ghost-OS/1.0")
        .redirect(reqwest::redirect::Policy::limited(5))
        .build()
        .map_err(|err| format!("failed to initialize HTTP client: {err}"))?;

    let response = client
        .get(url.clone())
        .send()
        .map_err(|err| format!("request failed: {err}"))?;

    if !response.status().is_success() {
        return Err(format!("request failed with status {}", response.status()));
    }

    if let Some(content_length) = response.content_length()
        && content_length > max_bytes as u64
    {
        return Err(format!("response too large (max {} bytes)", max_bytes));
    }

    let mut body = Vec::new();
    response
        .take(max_bytes as u64 + 1)
        .read_to_end(&mut body)
        .map_err(|err| format!("failed to read response body: {err}"))?;

    if body.len() > max_bytes {
        return Err(format!("response too large (max {} bytes)", max_bytes));
    }

    Ok(html2text::from_read(body.as_slice(), 100))
}

fn is_private_ip(ip: IpAddr) -> bool {
    match ip {
        IpAddr::V4(ipv4) => {
            let octets = ipv4.octets();
            let in_cgnat = octets[0] == 100 && (octets[1] & 0b1100_0000) == 0b0100_0000;
            ipv4.is_private()
                || ipv4.is_loopback()
                || ipv4.is_link_local()
                || ipv4.is_multicast()
                || ipv4.is_broadcast()
                || ipv4.is_documentation()
                || in_cgnat
                || octets[0] == 0
        }
        IpAddr::V6(ipv6) => {
            ipv6.is_loopback()
                || ipv6.is_unspecified()
                || ipv6.is_unique_local()
                || ipv6.is_multicast()
                || ipv6.is_unicast_link_local()
        }
    }
}
