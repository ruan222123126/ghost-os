# Web Rooter HTTP Integration

This document locks the Ghost-OS bridge contract for the external `web_rooter`
tool. The upstream reference is `baojiachen0214/web-rooter` pinned to `v0.2.4`
and source commit `77b0104137ed4b173ba19e6b5e3a23817ec5e198`.

## Deployment Boundary

- Ghost-OS only talks to an already-running `web-rooter` HTTP service via
  `web_rooter_base_url`.
- Development mode: start the upstream server manually, for example from the
  pinned checkout with `python server.py`.
- Production mode: run `web-rooter` as a separate container or service process.
- Ghost-OS does not install or manage upstream Python dependencies such as
  Playwright, FastAPI, or MCP packages.
- Ghost-OS does not spawn, supervise, or restart the sidecar process.

## Session Boundary

- Ghost-OS only exposes six stateless HTTP actions through the single
  `web_rooter` bridge tool:
  - `internet_search`
  - `research`
  - `academic_search`
  - `site_search`
  - `fetch`
  - `extract`
- Stateful upstream surfaces stay out of scope:
  - `knowledge`
  - `visited`
  - context snapshots
  - planner / jobs / workflow endpoints
- Ghost-OS session state remains the only session truth. The bridge must not
  read or write `web-rooter` global knowledge, visited URLs, or context state.

## Trace Contract

- Every bridge call forwards the current `trace_id` to the sidecar as the
  `X-Trace-ID` request header.
- The trace header is sent on the version probe and on the action request so
  bridge logs can correlate both legs of the sidecar call chain.

## Result Size Contract

- The bridge does not silently trim upstream research payloads.
- Callers must control result volume explicitly with the action parameters:
  - `num_results`
  - `max_pages`
  - `auto_crawl`
  - `fetch_abstracts`
  - `use_browser`
- If the upstream response is malformed, oversized, or otherwise invalid, the
  bridge returns an explicit error instead of falling back to another tool or
  mutating the payload.
