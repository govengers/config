# govengers/config: design

Decisions are numbered `C1` onwards; a later change gets a new number, never a rewrite. The v2 decisions taken before this file existed (global registry as the DES-003 exception, `Into`, every problem in one error, no partial state on failure, `Sensitive` out of errors) are described in the README and its migration table.

## C1 Every value type has its own builder

Each type is a builder of its own (`Duration`, `Float`, `URL`, `Enum` next to `String`, `StringArray`, `Bool`, `Int`) with the methods that make sense for it, so `Into` is typed and a wrong target fails to compile. `NotEmpty` exists only where a value can be empty and still parse: `String`, `StringArray` and `URL`. A `Duration`, `Float` or `Int` rejects an empty value already, and an `Enum` accepts the empty string only if it is listed.

## C2 Duration uses Go syntax and rejects a bare number

`time.ParseDuration` decides: `30s`, `5m`, `1h30m`, negative values included. A number without a unit fails (only `0` passes), because "300" meaning 300 nanoseconds is a known way to build a hot loop. Seconds as an `Int` stay possible for existing variables, but new ones use `Duration`.

## C3 Float rejects NaN and infinities

`strconv.ParseFloat` accepts `NaN` and `Inf`. Both are always a mistake in configuration and break every comparison downstream, so they are rejected at load, in defaults, flags and `Set`.

## C4 URL is parsed into url.URL, absolute with a host, with an optional scheme allowlist

- The stored value is a `url.URL`: `Into(*url.URL)` writes a copy into a value field. The getter returns a `*url.URL` to a fresh copy on every call, never nil, because every `url.URL` method has a pointer receiver and a returned struct is not addressable (`Get().URL(key).String()` would not compile). No caller can change another caller's URL; `Userinfo` is immutable, sharing it is safe.
- A non-empty value must have a scheme and a host name; a port alone (`http://:8080`) is no host. `url.Parse` accepts nearly anything: `localhost:8080` parses as scheme `localhost`, `api.example/v1` as a path. Requiring both catches the common mistakes. URLs without a host (Unix socket DSNs, `file:`) are declared as `String`.
- An empty value is the zero `url.URL`, like an empty `String`; `NotEmpty` rejects it. This keeps the CFG-004 pattern (`X_ENABLED` plus an optional URL with `Default("")`) working.
- `Schemes(...)` compares lowercased; `url.Parse` lowercases the scheme already. The empty URL is not checked against the allowlist.
- `Default` takes text, parsed like the environment variable, because a `url.URL` literal is unreadable at the declaration. A default that does not parse fails `LoadConfig`, even when the variable is set (same rule as an empty `NotEmpty` default).
- The error of `url.Parse` is dropped, because it quotes the value and the value may hold credentials. A URL that is not `Sensitive` never shows its password: `Print` and every error (environment, default, flag) quote `Redacted()`, and a value `url.URL` cannot parse is not quoted at all, because its password cannot be located. Declaring credential URLs `Sensitive` (CFG-002) still masks the rest.

## C5 Enum is a closed, case-sensitive set

`Enum(name, allowed...)` takes the members at declaration. Comparison is exact: accepting `INFO` for `info` would need a normalisation rule per caller, and the error lists the members, so the fix is obvious. An `Enum` without members is a declaration error. The registry type is `enum`: `Get().Enum(key)` reads it, `Get().String(key)` panics like every other type mismatch.

## C6 One rule check for every source of a value

The rules a declaration adds to its type (Enum members, URL schemes, finite floats) live in one function that runs on environment values, defaults, `BindPFlag` and `Set`. `BindPFlag` checks them while parsing, so its error names the flag and the rejected value like a parse error does. No path can store a value `LoadConfig` would reject, which extends the guarantee `NotEmpty` already had.
