package gaze

import _ "embed"

// Skill contains the bundled gaze CLI skill definition.
//
//go:embed skills/gaze-cli/SKILL.md
var Skill []byte
