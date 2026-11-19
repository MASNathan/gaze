package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/masnathan/gaze"
)

type skillAgent struct{ name, binary, project, global string }

func installSkill(args []string, input io.Reader, out, errout io.Writer) int {
	flags := flag.NewFlagSet("gaze skill", flag.ContinueOnError)
	flags.SetOutput(errout)
	agentsFlag := flags.String("agents", "", "comma-separated agents: claude,codex,opencode")
	scope := flags.String("scope", "", "installation scope: project or user")
	update := flags.Bool("update", false, "update existing installations without prompting")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(errout, "gaze skill: unexpected arguments")
		return 1
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(errout, "gaze skill:", err)
		return 1
	}
	config, err := os.UserConfigDir()
	if err != nil {
		fmt.Fprintln(errout, "gaze skill:", err)
		return 1
	}
	agents := []skillAgent{
		{"claude", "claude", ".claude/skills", filepath.Join(home, ".claude", "skills")},
		{"codex", "codex", ".agents/skills", filepath.Join(home, ".agents", "skills")},
		{"opencode", "opencode", ".opencode/skills", filepath.Join(config, "opencode", "skills")},
	}
	if *update {
		if *agentsFlag != "" || *scope != "" {
			fmt.Fprintln(errout, "gaze skill: -update cannot be combined with -agents or -scope")
			return 1
		}
		var paths []string
		for _, a := range agents {
			for _, dir := range []string{a.project, a.global} {
				path := filepath.Join(dir, "gaze-cli", "SKILL.md")
				info, err := os.Lstat(path)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					fmt.Fprintln(errout, "gaze skill:", err)
					return 1
				}
				if !info.Mode().IsRegular() {
					fmt.Fprintf(errout, "gaze skill: refusing non-regular file %s\n", path)
					return 1
				}
				paths = append(paths, path)
			}
		}
		if len(paths) == 0 {
			fmt.Fprintln(errout, "gaze skill: no existing installations found")
			return 1
		}
		for _, path := range paths {
			if err := writeSkill(path); err != nil {
				fmt.Fprintln(errout, "gaze skill:", err)
				return 1
			}
			fmt.Fprintf(out, "Updated: %s\n", path)
		}
		return 0
	}
	selected := map[string]bool{}
	if *agentsFlag != "" {
		for _, name := range strings.Split(*agentsFlag, ",") {
			selected[strings.TrimSpace(name)] = true
		}
	} else {
		var options []huh.Option[string]
		for _, a := range agents {
			if agentAvailable(a) {
				options = append(options, huh.NewOption(a.name, a.name))
			}
		}
		if len(options) == 0 {
			fmt.Fprintln(errout, "gaze skill: no agent executables found; specify -agents explicitly")
			return 1
		}
		var choices []string
		field := huh.NewMultiSelect[string]().Title("Select agents").Options(options...).Value(&choices).Validate(func(values []string) error {
			if len(values) == 0 {
				return fmt.Errorf("select at least one agent")
			}
			return nil
		})
		if err := huh.NewForm(huh.NewGroup(field)).WithInput(input).WithOutput(out).Run(); err != nil {
			fmt.Fprintln(errout, "gaze skill:", err)
			return 1
		}
		for _, choice := range choices {
			selected[choice] = true
		}
	}
	for name := range selected {
		found := false
		for _, a := range agents {
			if name == a.name {
				found = true
			}
		}
		if !found {
			fmt.Fprintf(errout, "gaze skill: unknown agent %q\n", name)
			return 1
		}
	}
	if *scope == "" {
		field := huh.NewSelect[string]().Title("Installation scope").Options(
			huh.NewOption("Project", "project"), huh.NewOption("User", "user"),
		).Value(scope)
		if err := huh.NewForm(huh.NewGroup(field)).WithInput(input).WithOutput(out).Run(); err != nil {
			fmt.Fprintln(errout, "gaze skill:", err)
			return 1
		}
	}
	if *scope != "project" && *scope != "user" {
		fmt.Fprintln(errout, "gaze skill: scope must be project or user")
		return 1
	}
	var paths []string
	for _, a := range agents {
		if selected[a.name] {
			dir := a.project
			if *scope == "user" {
				dir = a.global
			}
			path := filepath.Join(dir, "gaze-cli", "SKILL.md")
			paths = append(paths, path)
			fmt.Fprintf(out, "%s: %s\n", a.name, path)
		}
	}
	// Refuse to replace non-regular files, including symlinks.
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				fmt.Fprintf(errout, "gaze skill: refusing non-regular file %s\n", path)
				return 1
			}
		} else if !os.IsNotExist(err) {
			fmt.Fprintln(errout, "gaze skill:", err)
			return 1
		}
	}
	if *agentsFlag == "" || !flagProvided(args, "scope") {
		confirmed := false
		field := huh.NewConfirm().Title("Install or replace these skills?").Value(&confirmed)
		if err := huh.NewForm(huh.NewGroup(field)).WithInput(input).WithOutput(out).Run(); err != nil {
			fmt.Fprintln(errout, "gaze skill:", err)
			return 1
		}
		if !confirmed {
			fmt.Fprintln(out, "Cancelled.")
			return 1
		}
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			fmt.Fprintln(errout, "gaze skill:", err)
			return 1
		}
		if err := writeSkill(path); err != nil {
			fmt.Fprintln(errout, "gaze skill:", err)
			return 1
		}
	}
	return 0
}

func writeSkill(path string) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".gaze-skill-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0644); err == nil {
		_, err = f.Write(gaze.Skill)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	return err
}

func flagProvided(args []string, name string) bool {
	for _, a := range args {
		if a == "-"+name || a == "--"+name || strings.HasPrefix(a, "-"+name+"=") || strings.HasPrefix(a, "--"+name+"=") {
			return true
		}
	}
	return false
}

func agentAvailable(a skillAgent) bool {
	if _, err := exec.LookPath(a.binary); err == nil {
		return true
	}
	if a.name == "opencode" {
		_, err := exec.LookPath("opencode2")
		return err == nil
	}
	return false
}
