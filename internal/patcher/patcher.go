package patcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/cybellereaper/wizturtle/v2/internal/manifest"
	"github.com/cybellereaper/wizturtle/v2/internal/state"
)

// ErrUpToDate is returned when no files need to be patched.
var ErrUpToDate = errors.New("patcher: already up to date")

// Logger defines the logging contract expected by the patcher.
type Logger interface {
	Printf(format string, args ...any)
}

// Config configures a patcher instance.
type Config struct {
	PatchInfoURL string
	InstallDir   string
	StateStore   *state.Store
	HTTPClient   *http.Client
	Concurrency  int
	Logger       Logger
}

// PatchInfo describes the response served by the patch metadata endpoint.
type PatchInfo struct {
	Version     string `json:"version"`
	ManifestURL string `json:"manifest"`
	BaseURL     string `json:"base_url,omitempty"`
}

// Patcher coordinates fetching manifests and updating an installation.
type Patcher struct {
	cfg      Config
	logger   Logger
	client   *http.Client
	store    *state.Store
	manifest manifest.Manifest
	info     PatchInfo
}

// New creates a configured patcher instance.
func New(cfg Config) (*Patcher, error) {
	if strings.TrimSpace(cfg.PatchInfoURL) == "" {
		return nil, errors.New("patcher: patch info URL is required")
	}
	if strings.TrimSpace(cfg.InstallDir) == "" {
		return nil, errors.New("patcher: install directory is required")
	}
	if cfg.StateStore == nil {
		return nil, errors.New("patcher: state store is required")
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	logger := cfg.Logger
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	return &Patcher{
		cfg: func() Config {
			concurrency := cfg.Concurrency
			if concurrency <= 0 {
				concurrency = runtime.NumCPU()
				if concurrency < 2 {
					concurrency = 2
				}
			}
			cfg.Concurrency = concurrency
			cfg.HTTPClient = client
			cfg.Logger = logger
			return cfg
		}(),
		logger: logger,
		client: client,
		store:  cfg.StateStore,
	}, nil
}

// Run executes the patching workflow end-to-end.
func (p *Patcher) Run(ctx context.Context) error {
	snapshot, err := p.store.Load(ctx)
	if err != nil {
		return err
	}

	info, err := p.fetchPatchInfo(ctx)
	if err != nil {
		return err
	}
	p.info = info

	manifestBytes, err := p.download(ctx, info.ManifestURL)
	if err != nil {
		return fmt.Errorf("patcher: manifest download: %w", err)
	}

	parsedManifest, err := manifest.Parse(manifestBytes)
	if err != nil {
		return err
	}
	p.manifest = parsedManifest

	base, err := p.manifestBaseURL()
	if err != nil {
		return err
	}

	toDownload := make([]manifest.File, 0, len(parsedManifest.Files))
	newState := state.Snapshot{Version: parsedManifest.Version, Files: make(map[string]state.FileInfo, len(parsedManifest.Files))}

	for _, entry := range parsedManifest.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		dest := filepath.Join(p.cfg.InstallDir, filepath.FromSlash(entry.Target))
		valid, meta, err := validateLocalFile(dest, entry)
		if err != nil {
			p.logger.Printf("invalid file %s: %v", entry.Target, err)
		}
		if valid {
			newState.Files[entry.Target] = meta
			continue
		}

		toDownload = append(toDownload, entry)
	}

	if len(toDownload) == 0 && snapshot.Version == parsedManifest.Version {
		if err := p.store.Save(ctx, newState); err != nil {
			return err
		}
		return ErrUpToDate
	}

	p.logger.Printf("Starting patch to version %s (%d files to update)", parsedManifest.Version, len(toDownload))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(p.effectiveConcurrency())

	var mu sync.Mutex

	for _, entry := range toDownload {
		entry := entry
		g.Go(func() error {
			meta, err := p.downloadEntry(gctx, base, entry)
			if err != nil {
				return err
			}
			mu.Lock()
			newState.Files[entry.Target] = meta
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Fill in metadata for files that were already valid but not part of downloads.
	for _, entry := range parsedManifest.Files {
		if _, ok := newState.Files[entry.Target]; ok {
			continue
		}
		dest := filepath.Join(p.cfg.InstallDir, filepath.FromSlash(entry.Target))
		_, meta, err := validateLocalFile(dest, entry)
		if err != nil {
			return fmt.Errorf("patcher: expected %s to be valid after download: %w", entry.Target, err)
		}
		newState.Files[entry.Target] = meta
	}

	if err := p.store.Save(ctx, newState); err != nil {
		return err
	}

	p.logger.Printf("Patch completed successfully")
	return nil
}

func (p *Patcher) effectiveConcurrency() int {
	if p.cfg.Concurrency > 0 {
		return p.cfg.Concurrency
	}
	return runtime.NumCPU()
}

func (p *Patcher) fetchPatchInfo(ctx context.Context) (PatchInfo, error) {
	body, err := p.download(ctx, p.cfg.PatchInfoURL)
	if err != nil {
		return PatchInfo{}, fmt.Errorf("patcher: patch info download: %w", err)
	}

	var info PatchInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return PatchInfo{}, fmt.Errorf("patcher: decode patch info: %w", err)
	}

	if strings.TrimSpace(info.Version) == "" {
		return PatchInfo{}, errors.New("patcher: patch info missing version")
	}
	if strings.TrimSpace(info.ManifestURL) == "" {
		return PatchInfo{}, errors.New("patcher: patch info missing manifest URL")
	}

	return info, nil
}

func (p *Patcher) download(ctx context.Context, resource string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resource, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, resource)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (p *Patcher) manifestBaseURL() (*url.URL, error) {
	if strings.TrimSpace(p.info.BaseURL) != "" {
		return url.Parse(p.info.BaseURL)
	}
	return url.Parse(p.info.ManifestURL)
}

func (p *Patcher) downloadEntry(ctx context.Context, base *url.URL, entry manifest.File) (state.FileInfo, error) {
	dest := filepath.Join(p.cfg.InstallDir, filepath.FromSlash(entry.Target))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return state.FileInfo{}, fmt.Errorf("mkdir %s: %w", dest, err)
	}

	targetURL, err := resolveURL(base, entry.Source)
	if err != nil {
		return state.FileInfo{}, fmt.Errorf("resolve %s: %w", entry.Source, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return state.FileInfo{}, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return state.FileInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return state.FileInfo{}, fmt.Errorf("download %s: status %d", entry.Source, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".wizturtle-*")
	if err != nil {
		return state.FileInfo{}, err
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body)
	if err != nil {
		return state.FileInfo{}, err
	}

	if written != entry.Size {
		return state.FileInfo{}, fmt.Errorf("size mismatch for %s: expected %d, got %d", entry.Target, entry.Size, written)
	}

	digest := hex.EncodeToString(hasher.Sum(nil))
	if digest != entry.SHA256 {
		return state.FileInfo{}, fmt.Errorf("hash mismatch for %s", entry.Target)
	}

	if err := tmp.Close(); err != nil {
		return state.FileInfo{}, err
	}

	if err := os.Rename(tmp.Name(), dest); err != nil {
		return state.FileInfo{}, err
	}

	if entry.Mode != 0 {
		if err := os.Chmod(dest, entry.Mode); err != nil {
			return state.FileInfo{}, err
		}
	}

	return state.FileInfo{Size: written, SHA256: digest}, nil
}

func resolveURL(base *url.URL, path string) (*url.URL, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("empty path")
	}
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	if u.IsAbs() {
		return u, nil
	}
	return base.ResolveReference(u), nil
}

func validateLocalFile(path string, entry manifest.File) (bool, state.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, state.FileInfo{}, nil
		}
		return false, state.FileInfo{}, err
	}

	if info.Size() != entry.Size {
		return false, state.FileInfo{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return false, state.FileInfo{}, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false, state.FileInfo{}, err
	}

	digest := hex.EncodeToString(hasher.Sum(nil))
	if digest != entry.SHA256 {
		return false, state.FileInfo{}, nil
	}

	return true, state.FileInfo{Size: entry.Size, SHA256: digest}, nil
}
