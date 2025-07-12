package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"go.uber.org/zap"
	"sigs.k8s.io/yaml"
)

type GitClient struct {
	config   *config.GitConfig
	repoPath string
	logger   *zap.Logger
}

type GitChange struct {
	CommitHash string
	Author     string
	Message    string
	Files      []string
	Timestamp  int64
}

type GitCommit struct {
	Hash      string
	Author    string
	Message   string
	Timestamp int64
	Files     []string
}

type GitStatus struct {
	Branch     string
	CommitHash string
	IsClean    bool
	Modified   []string
	Untracked  []string
}

func NewGitClient(cfg *config.GitConfig, logger *zap.Logger) (*GitClient, error) {
	repoPath, err := os.MkdirTemp("", "driftguard-git-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &GitClient{
		config:   cfg,
		repoPath: repoPath,
		logger:   logger,
	}, nil
}

func (g *GitClient) Clone() error {
	g.logger.Info("Cloning Git repository", zap.String("path", g.repoPath))

	ctx, cancel := context.WithTimeout(context.Background(), g.config.CloneTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", g.config.DefaultBranch, g.config.RepositoryURL, g.repoPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	g.logger.Info("Successfully cloned repository")
	return nil
}

func (g *GitClient) Pull() error {
	g.logger.Debug("Pulling latest changes from Git repository")

	ctx, cancel := context.WithTimeout(context.Background(), g.config.PullTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "pull", "origin", g.config.DefaultBranch)
	cmd.Dir = g.repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull repository: %w", err)
	}

	g.logger.Debug("Successfully pulled latest changes")
	return nil
}

func (g *GitClient) GetFile(path string) ([]byte, error) {
	fullPath := filepath.Join(g.repoPath, path)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	return os.ReadFile(fullPath)
}

func (g *GitClient) GetK8sManifests() ([]map[string]interface{}, error) {
	g.logger.Debug("Extracting Kubernetes manifests from Git repository")

	var manifests []map[string]interface{}

	err := filepath.Walk(g.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !isYAMLFile(path) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			g.logger.Warn("Failed to read file", zap.String("path", path), zap.Error(err))
			return nil
		}

		parsedManifests, err := g.parseYAMLContent(data)
		if err != nil {
			g.logger.Warn("Failed to parse YAML file", zap.String("path", path), zap.Error(err))
			return nil
		}

		for _, manifest := range parsedManifests {
			if g.isKubernetesResource(manifest) {
				manifests = append(manifests, manifest)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk repository: %w", err)
	}

	g.logger.Info("Extracted Kubernetes manifests", zap.Int("count", len(manifests)))
	return manifests, nil
}

func (g *GitClient) GetManifestForResource(kind, namespace, name string) (map[string]interface{}, error) {
	manifests, err := g.GetK8sManifests()
	if err != nil {
		return nil, err
	}

	for _, manifest := range manifests {
		manifestKind := getStringValue(manifest, "kind")
		manifestNamespace := getStringValue(manifest, "metadata.namespace")
		manifestName := getStringValue(manifest, "metadata.name")

		if manifestKind == kind && manifestNamespace == namespace && manifestName == name {
			return manifest, nil
		}
	}

	return nil, fmt.Errorf("manifest not found for resource %s/%s/%s", kind, namespace, name)
}

func (g *GitClient) WatchChanges() (<-chan *GitChange, error) {
	changeCh := make(chan *GitChange)
	return changeCh, nil
}

func (g *GitClient) GetCommitHistory(path string) ([]*GitCommit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "--follow", path)
	cmd.Dir = g.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	var commits []*GitCommit
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		commits = append(commits, &GitCommit{
			Hash:    parts[0],
			Message: parts[1],
		})
	}

	return commits, nil
}

func (g *GitClient) ValidateManifests() error {
	manifests, err := g.GetK8sManifests()
	if err != nil {
		return err
	}

	for _, manifest := range manifests {
		if err := g.validateManifest(manifest); err != nil {
			return fmt.Errorf("invalid manifest: %w", err)
		}
	}

	return nil
}

func (g *GitClient) SwitchBranch(branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "checkout", branch)
	cmd.Dir = g.repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (g *GitClient) GetCurrentCommit() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = g.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (g *GitClient) GetFileDiff(filePath, fromCommit, toCommit string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", fromCommit, toCommit, "--", filePath)
	cmd.Dir = g.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get file diff: %w", err)
	}

	return string(output), nil
}

func (g *GitClient) IsRepositoryClean() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = g.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check repository status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) == 0, nil
}

func (g *GitClient) GetRepositoryStatus() (*GitStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	branchCmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = g.repoPath
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	commitCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	commitCmd.Dir = g.repoPath
	commitOutput, err := commitCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit: %w", err)
	}

	isClean, err := g.IsRepositoryClean()
	if err != nil {
		return nil, err
	}

	return &GitStatus{
		Branch:     strings.TrimSpace(string(branchOutput)),
		CommitHash: strings.TrimSpace(string(commitOutput)),
		IsClean:    isClean,
	}, nil
}

func (g *GitClient) Cleanup() error {
	return os.RemoveAll(g.repoPath)
}

func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func (g *GitClient) parseYAMLContent(data []byte) ([]map[string]interface{}, error) {
	documents := strings.Split(string(data), "---")
	var manifests []map[string]interface{}

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			g.logger.Warn("Failed to unmarshal YAML document", zap.Error(err))
			continue
		}

		if len(manifest) > 0 {
			manifests = append(manifests, manifest)
		}
	}

	return manifests, nil
}

func (g *GitClient) isKubernetesResource(manifest map[string]interface{}) bool {
	apiVersion := getStringValue(manifest, "apiVersion")
	kind := getStringValue(manifest, "kind")
	metadata := getMapValue(manifest, "metadata")

	return apiVersion != "" && kind != "" && metadata != nil
}

func (g *GitClient) validateManifest(manifest map[string]interface{}) error {
	apiVersion := getStringValue(manifest, "apiVersion")
	kind := getStringValue(manifest, "kind")
	metadata := getMapValue(manifest, "metadata")

	if apiVersion == "" {
		return fmt.Errorf("missing apiVersion")
	}
	if kind == "" {
		return fmt.Errorf("missing kind")
	}
	if metadata == nil {
		return fmt.Errorf("missing metadata")
	}

	return nil
}

func getStringValue(data map[string]interface{}, path string) string {
	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		if val, ok := current[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
			if mapVal, ok := val.(map[string]interface{}); ok {
				current = mapVal
			} else {
				return ""
			}
		} else {
			return ""
		}
	}

	return ""
}

func getMapValue(data map[string]interface{}, path string) map[string]interface{} {
	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		if val, ok := current[key]; ok {
			if mapVal, ok := val.(map[string]interface{}); ok {
				current = mapVal
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}
