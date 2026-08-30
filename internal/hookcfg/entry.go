// Package hookcfg reconciles co-owned agent hook configuration files
// (.claude/settings.json, .codex/hooks.json) against forge's canonical
// embedded template without blind-overwriting entries other tools have
// added.
//
// Both files share one schema: a top-level "hooks" object mapping an event
// name (e.g. "SessionStart", "PreToolUse") to an array of matcher groups
// ({"matcher": string, "hooks": [{"type":"command","command":string}]}).
// forge owns exactly the entries present in its canonical template, keyed by
// command string. Everything else — third-party matcher groups, entries
// inside events forge does not declare, and unrelated top-level keys — is
// foreign and must survive reconciliation untouched.
package hookcfg

// Registry declares forge's entry-identity rules for one co-owned file.
//
// Historical maps a current owned command (a command string that appears in
// the canonical template) to prior shipped forms of that same command. When
// Reconcile finds a Historical fingerprint living inside an event the
// canonical template owns, it claims that entry and converges it to the
// current command in place — never as a duplicate alongside the current
// form. A Historical fingerprint that only appears inside an event the
// canonical template does not own is left untouched: ownership claiming is
// scoped per owning event, not global string matching.
type Registry struct {
	Historical map[string][]string
}

// canonicalCommand returns the command string carried by a hook entry
// object, and whether the entry was shaped as expected
// ({"type":"command","command":"..."} or any object with a string
// "command" field).
func entryCommand(entry any) (string, bool) {
	m, ok := entry.(map[string]any)
	if !ok {
		return "", false
	}
	cmd, ok := m["command"].(string)
	return cmd, ok
}

// groupHooks returns a matcher group's "hooks" array.
func groupHooks(group any) ([]any, bool) {
	m, ok := group.(map[string]any)
	if !ok {
		return nil, false
	}
	hooks, ok := m["hooks"].([]any)
	return hooks, ok
}

// groupMatcher returns a matcher group's "matcher" string.
func groupMatcher(group any) (string, bool) {
	m, ok := group.(map[string]any)
	if !ok {
		return "", false
	}
	matcher, ok := m["matcher"].(string)
	return matcher, ok
}

// newCommandEntry builds a canonical-shaped hook entry for an owned command.
func newCommandEntry(command string) map[string]any {
	return map[string]any{"type": "command", "command": command}
}
