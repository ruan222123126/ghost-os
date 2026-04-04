# GraphQL Tool Runtime

Ghost-OS Bridge now supports a dedicated GraphQL tool-call protocol mode.

When `graphql_tool_runtime_enabled = true`, the model no longer receives native
structured tool definitions and must reply with exactly one GraphQL document.
Bridge parses that document, resolves the top-level field to a real bridge
tool, executes the tool through the normal tool execution pipeline, and feeds
the result back into the next completion round.

## Configuration

Enable the runtime in `config.toml`:

```toml
graphql_tool_runtime_enabled = true
```

Default behavior:

- `false`: normal native tool-calling mode
- `true`: GraphQL tool runtime mode only

This mode does not run in parallel with native `tool_calls`.

## Protocol

The model must return exactly one GraphQL document and use one or more
`mutation` operations. Each operation must contain exactly one top-level field:

```graphql
mutation {
  web_search(query: "OpenAI latest news", max_results: 5)
}
```

```graphql
mutation {
  script_exec(script: "print('hello')")
}
```

Rules:

- One or more operations per document
- Exactly one top-level field per operation
- Operations execute in written order
- Field name must exactly match a visible tool name
- Arguments are mapped directly to the tool JSON parameter object
- Use `mutation` for every tool call
- `query` is treated as a protocol error, not as a valid capability split
- A `tfind(action: load)` operation can make tools available to later operations in the same document

Unsupported features:

- aliases
- fragments
- variables
- directives
- nested selections
- multiple top-level fields

Protocol violations return explicit errors. Bridge does not silently fall back
to plain assistant text or native tool-calling.

## Runtime behavior

When GraphQL tool runtime is enabled:

- completion request `tools` is empty, so the model cannot see native tool defs
- the system prompt is augmented with a GraphQL tool schema summary built from
  the current turn's visible tool catalog
- `assistant_text_graphql` is treated as a GraphQL tool-call document handler,
  not as a business GraphQL source executor
- parsed tool calls reuse the normal tool execution pipeline, including:
  - argument validation
  - `tool_call_id` injection
  - streaming `tool_call_started` / `tool_call_finished`
  - `awaiting_human`
  - tool result envelope writing
  - internal feedback for the next completion round

Successful internal feedback uses the `[TOOL_TAG_RESULT]` prefix.

## Persistence

Bridge persists GraphQL tool runtime turns the same way as normal tool turns:

- the assistant message keeps the original GraphQL document
- tool outputs are written as standard `role=tool` messages
- internal feedback is added for the next completion round

If a later completion in the same turn fails, already executed assistant/tool
messages from that turn are still committed.
