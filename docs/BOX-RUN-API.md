# Platform-started box runs (`StartBoxRun`)

`StartBoxRun` (`POST /v1/containers/{username}/code-runs`, see `proto/containarium/v1/box_run.proto`)
starts or resumes a coding-agent run on a box through the daemon, with no SSH
client. The run leaves the same on-disk contract `containarium code run` leaves
in the box's run directory (`/tmp/agent-box`), so `ListBoxRuns`, `TailBoxRunLog`
and `containarium code runs|attach|status|logs` read it the same way. This page
covers what a platform caller (a control plane starting runs for members) needs
beyond the CLI.

## Naming a run: `run_name`

By default a run is named after the session id when resuming, else `code`.
A caller that cannot rely on `code` being free sets `run_name`:

- The run is named exactly that. The name must match `[A-Za-z0-9._-]{1,128}`
  and not be `.` or `..`; anything else is `INVALID_ARGUMENT` before anything
  runs on the box.
- The usual rules apply to that name: a finished run of the same name is
  rotated aside, a `RUNNING` or `UNKNOWN` one is `FAILED_PRECONDITION`.
- A named run never collides with the default `code` run, so a member's own
  `containarium code run` is unaffected. Pick a unique name per run to keep
  every log (a reused name rotates the previous log aside before the new run
  starts).
- With `session_id` also set, the run resumes that session but keeps
  `run_name` as its name.

The CLI counterpart is the existing `containarium code run <box> --name <name>`
(the SSH path); there is no separate flag.

## Structured output: `structured_output`

With `structured_output: true` (Claude Code boxes only; other engines are
`FAILED_PRECONDITION`) the engine runs with `--output-format json`:

| File | Content |
| --- | --- |
| `<run_name>.log` | Exactly the engine's one JSON result object: `type` (`result`), `subtype` (for example `success` or an error subtype), `is_error`, `result` (the final text), `session_id`, `usage`, and the engine's other result fields. |
| `<run_name>.stderr.log` | The engine's stderr, kept apart so the main log stays valid JSON. |

The object is written when the run ends; while it runs the main log is empty.
`TailBoxRunLog` and `containarium code logs` return the log as opaque bytes, so
they return the object verbatim. When the run exits, the object's `session_id`
is written to the run's session sidecar, before the exit record, so
`BoxRun.session_id` is set as soon as the run is `EXITED` (it replaces the
literal `session_id` of a resumed run, since the engine's reported id is the
one to resume next). If the object has no usable `session_id` the field stays
empty. The default (`false`) leaves plain-text behaviour unchanged.

## Session id of a fresh run

Without `structured_output`, the id of a fresh run is found by looking at the
engine's session directory. To avoid adopting an older conversation, the daemon
lists that directory just before launching the engine, then polls (up to about
12 seconds) for an entry that was not in the listing. Exactly one new entry is
the run's session. No new entry, or more than one (another conversation started
in the same window), leaves `session_id` empty; an older entry is never used as
a fallback. A caller that needs the id reliably should use `structured_output`.
A resumed run's id is the `session_id` it was started with.

## Feature detection

The REST gateway rejects request fields it does not know, so a caller must not
send `run_name` or `structured_output` to a daemon that predates them.
`GetSystemInfo` (`SystemInfo.capabilities`) lists the optional features a daemon
supports: `box_run_name` and `box_run_structured_output`. An older daemon does
not send the field at all; treat a missing name as unsupported. Names are only
added. `GetSystemInfo` requires an admin principal. A caller that cannot read it
can gate on the daemon version instead: the first release that contains these
fields (see the changelog entry).
