package cctidy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// SkillNameSet is a set of known skill names.
type SkillNameSet map[string]bool

// LoadSkillNames scans the skills and commands directories under
// claudeDir and returns a set of skill names.
//
// Skills are identified by subdirectories in <claudeDir>/skills/
// that contain a SKILL.md file. The subdirectory name is the skill name.
//
// Commands are identified by .md files in <claudeDir>/commands/.
// The filename without extension is the skill name.
//
// Returns an empty set if claudeDir is empty or unreadable.
func LoadSkillNames(claudeDir string) SkillNameSet {
	set := make(SkillNameSet)
	if claudeDir == "" {
		return set
	}
	loadSkillsDir(filepath.Join(claudeDir, "skills"), set)
	loadCommandsDir(filepath.Join(claudeDir, "commands"), set)
	return set
}

// loadSkillsDir scans dir for subdirectories containing SKILL.md.
func loadSkillsDir(dir string, set SkillNameSet) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			set[e.Name()] = true
		}
	}
}

// loadCommandsDir scans dir for .md files.
func loadCommandsDir(dir string, set SkillNameSet) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".md" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		if name != "" {
			set[name] = true
		}
	}
}

// SkillToolSweeper sweeps Skill permission entries where the
// referenced skill or command no longer exists. Plugin skills
// (containing ":") are always kept.
type SkillToolSweeper struct {
	skills SkillNameSet
}

// NewSkillToolSweeper creates a SkillToolSweeper.
func NewSkillToolSweeper(skills SkillNameSet) *SkillToolSweeper {
	return &SkillToolSweeper{skills: skills}
}

func (s *SkillToolSweeper) ShouldSweep(_ context.Context, entry StandardEntry) ToolSweepResult {
	specifier := entry.Specifier
	// Plugin skills use "plugin:name" convention
	// and are managed by the plugin system.
	if strings.Contains(specifier, ":") {
		return ToolSweepResult{}
	}
	// Extract name from specifier (e.g. "name *" -> "name").
	name, _, _ := strings.Cut(specifier, " ")
	if len(s.skills) == 0 {
		return ToolSweepResult{}
	}
	if s.skills[name] {
		return ToolSweepResult{}
	}
	return ToolSweepResult{Sweep: true}
}
