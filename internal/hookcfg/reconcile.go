package hookcfg

import (
	"encoding/json"
	"fmt"
)

// Reconcile merges canonical (forge's embedded template bytes) into existing
// (the on-disk co-owned file), converging forge's owned entries to their
// canonical form while leaving every third-party entry, group, and event
// untouched.
//
// Reconcile is pure: no file I/O, stdlib only. It never guesses at
// malformed input — a JSON parse failure on either input returns an error
// and writes nothing.
func Reconcile(existing, canonical []byte, reg Registry) (out []byte, changed bool, err error) {
	var existingDoc map[string]any
	if err := json.Unmarshal(existing, &existingDoc); err != nil {
		return nil, false, fmt.Errorf("hookcfg: parse existing file: %w", err)
	}

	var canonicalDoc map[string]any
	if err := json.Unmarshal(canonical, &canonicalDoc); err != nil {
		return nil, false, fmt.Errorf("hookcfg: parse canonical template: %w", err)
	}

	outDoc := make(map[string]any, len(existingDoc))
	for k, v := range existingDoc {
		outDoc[k] = v
	}

	existingHooks, _ := existingDoc["hooks"].(map[string]any)
	canonicalHooks, _ := canonicalDoc["hooks"].(map[string]any)

	newHooks := make(map[string]any, len(existingHooks))
	for event, groups := range existingHooks {
		// Copy every existing event verbatim first. Events canonical also
		// owns are overwritten below; unknown events pass through as-is.
		newHooks[event] = groups
	}

	docChanged := false

	for event, canonicalGroupsAny := range canonicalHooks {
		canonicalGroups, _ := canonicalGroupsAny.([]any)

		existingGroupsAny, present := existingHooks[event]
		if !present {
			// The whole owned event is missing from the existing file:
			// restore it wholesale from canonical.
			restored := make([]any, len(canonicalGroups))
			copy(restored, canonicalGroups)
			newHooks[event] = restored
			docChanged = true
			continue
		}

		existingGroups, _ := existingGroupsAny.([]any)
		mergedGroups, eventChanged := reconcileEvent(existingGroups, canonicalGroups, reg)
		newHooks[event] = mergedGroups
		if eventChanged {
			docChanged = true
		}
	}

	outDoc["hooks"] = newHooks

	// Non-hooks top-level keys: the existing file's value always wins;
	// canonical only supplies a key that is entirely absent.
	for k, v := range canonicalDoc {
		if k == "hooks" {
			continue
		}
		if _, present := existingDoc[k]; !present {
			outDoc[k] = v
			docChanged = true
		}
	}

	buf, err := json.MarshalIndent(outDoc, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("hookcfg: marshal reconciled document: %w", err)
	}
	buf = append(buf, '\n')

	return buf, docChanged, nil
}

// reconcileEvent merges one event's existing matcher groups against its
// canonical matcher groups. Owned entries (identified by command string,
// including any Historical fingerprints) are converged in place; missing
// owned entries are appended into the existing group that already carries a
// sibling owned entry from the same canonical group, or as a new group if no
// such group exists yet. Every unclaimed entry and group passes through
// untouched, in its original relative order.
func reconcileEvent(existingGroups, canonicalGroups []any, reg Registry) (out []any, changed bool) {
	type canonicalGroupInfo struct {
		matcher  string
		commands []string
	}

	ownedCommands := map[string]bool{}
	historicalToCanonical := map[string]string{}
	canonicalInfos := make([]canonicalGroupInfo, 0, len(canonicalGroups))

	for _, g := range canonicalGroups {
		matcher, _ := groupMatcher(g)
		hooks, _ := groupHooks(g)
		info := canonicalGroupInfo{matcher: matcher}
		for _, h := range hooks {
			cmd, ok := entryCommand(h)
			if !ok {
				continue
			}
			ownedCommands[cmd] = true
			info.commands = append(info.commands, cmd)
			for _, hist := range reg.Historical[cmd] {
				historicalToCanonical[hist] = cmd
			}
		}
		canonicalInfos = append(canonicalInfos, info)
	}

	claimed := map[string]bool{}
	outGroups := make([]any, len(existingGroups))
	// groupOwnedCmds[i] records which canonical-owned commands ended up
	// living in outGroups[i], so missing entries can be appended back into
	// the same group they belong with rather than a fresh one.
	groupOwnedCmds := make([]map[string]bool, len(existingGroups))

	for gi, g := range existingGroups {
		gm, ok := g.(map[string]any)
		if !ok {
			outGroups[gi] = g
			continue
		}

		hooksSlice, _ := gm["hooks"].([]any)
		newHooksSlice := make([]any, len(hooksSlice))
		ownedHere := map[string]bool{}

		for hi, h := range hooksSlice {
			cmd, ok := entryCommand(h)
			if !ok {
				newHooksSlice[hi] = h
				continue
			}

			if canonicalCmd, isHistorical := historicalToCanonical[cmd]; isHistorical {
				newHooksSlice[hi] = newCommandEntry(canonicalCmd)
				claimed[canonicalCmd] = true
				ownedHere[canonicalCmd] = true
				changed = true
				continue
			}

			if ownedCommands[cmd] {
				newHooksSlice[hi] = h
				claimed[cmd] = true
				ownedHere[cmd] = true
				continue
			}

			// Third-party entry: passes through untouched.
			newHooksSlice[hi] = h
		}

		newGroup := make(map[string]any, len(gm))
		for k, v := range gm {
			newGroup[k] = v
		}
		newGroup["hooks"] = newHooksSlice

		outGroups[gi] = newGroup
		groupOwnedCmds[gi] = ownedHere
	}

	for _, info := range canonicalInfos {
		var missing []string
		for _, cmd := range info.commands {
			if !claimed[cmd] {
				missing = append(missing, cmd)
			}
		}
		if len(missing) == 0 {
			continue
		}

		targetIdx := -1
		for gi := range outGroups {
			for _, cmd := range info.commands {
				if groupOwnedCmds[gi][cmd] {
					targetIdx = gi
					break
				}
			}
			if targetIdx != -1 {
				break
			}
		}

		if targetIdx == -1 {
			hooksList := make([]any, 0, len(missing))
			for _, cmd := range missing {
				hooksList = append(hooksList, newCommandEntry(cmd))
			}
			outGroups = append(outGroups, map[string]any{
				"matcher": info.matcher,
				"hooks":   hooksList,
			})
			for _, cmd := range missing {
				claimed[cmd] = true
			}
			changed = true
			continue
		}

		targetGroup := outGroups[targetIdx].(map[string]any)
		hooksList, _ := targetGroup["hooks"].([]any)
		for _, cmd := range missing {
			hooksList = append(hooksList, newCommandEntry(cmd))
			claimed[cmd] = true
		}
		targetGroup["hooks"] = hooksList
		outGroups[targetIdx] = targetGroup
		changed = true
	}

	return outGroups, changed
}
