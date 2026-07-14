package config

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// LabelConfig is the persisted org/repo -> labels mapping. It is not yet
// wired into the UI; it exists so a future "filter repos by label" step can
// be added without changing the on-disk schema or the org/repo identifiers
// used elsewhere in the app (they match wizard.State's org login / repo name).
type LabelConfig struct {
	Version int                  `yaml:"version"`
	Orgs    map[string]OrgLabels `yaml:"orgs"`
}

// OrgLabels holds the repos labeled within a single org.
type OrgLabels struct {
	Repos map[string]RepoLabels `yaml:"repos"`
}

// RepoLabels holds the labels assigned to a single repo.
type RepoLabels struct {
	Labels []string `yaml:"labels"`
}

// LoadLabels reads a LabelConfig from path. A missing file is not an error;
// it returns an empty, usable config so callers don't need to special-case
// "no config yet".
func LoadLabels(path string) (*LabelConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &LabelConfig{Version: 1, Orgs: map[string]OrgLabels{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading label config %s: %w", path, err)
	}

	var cfg LabelConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing label config %s: %w", path, err)
	}
	if cfg.Orgs == nil {
		cfg.Orgs = map[string]OrgLabels{}
	}
	return &cfg, nil
}

// Save writes the LabelConfig to path as YAML.
func (c *LabelConfig) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling label config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing label config %s: %w", path, err)
	}
	return nil
}

// LabelsFor returns the labels assigned to a repo, or nil if none.
func (c *LabelConfig) LabelsFor(org, repo string) []string {
	return c.Orgs[org].Repos[repo].Labels
}

// SetLabels assigns labels to a repo, creating intermediate maps as needed.
func (c *LabelConfig) SetLabels(org, repo string, labels []string) {
	if c.Orgs == nil {
		c.Orgs = map[string]OrgLabels{}
	}
	orgLabels, ok := c.Orgs[org]
	if !ok || orgLabels.Repos == nil {
		orgLabels = OrgLabels{Repos: map[string]RepoLabels{}}
	}
	orgLabels.Repos[repo] = RepoLabels{Labels: labels}
	c.Orgs[org] = orgLabels
}

// ReposByLabel returns the names of repos in org tagged with label, sorted.
func (c *LabelConfig) ReposByLabel(org, label string) []string {
	var repos []string
	for repo, rl := range c.Orgs[org].Repos {
		for _, l := range rl.Labels {
			if l == label {
				repos = append(repos, repo)
				break
			}
		}
	}
	sort.Strings(repos)
	return repos
}
