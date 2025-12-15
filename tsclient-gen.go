package httprpc

import (
	"os"
	"path/filepath"
	"strings"
)

// TSClientGenConfig configures TypeScript client generation.
type TSClientGenConfig struct {
	// Dir is the output directory for generated TypeScript.
	// If empty, generation is disabled.
	Dir string

	// Options are passed through to GenTSDir.
	Options TSGenOptions

	// OnError is called if GenerateTSClient encounters an error.
	// If nil, errors are only returned to the caller.
	OnError func(error)
}

// SetTSClientGenConfig sets the configuration for TypeScript client generation.
func (r *Router) SetTSClientGenConfig(cfg *TSClientGenConfig) {
	r.tsGenMu.Lock()
	defer r.tsGenMu.Unlock()

	if cfg == nil || strings.TrimSpace(cfg.Dir) == "" {
		r.tsGenCfg = nil
		return
	}
	c := *cfg
	c.Options = c.Options.withDefaults()
	r.tsGenCfg = &c
}

func (r *Router) tsClientGenConfig() *TSClientGenConfig {
	r.tsGenMu.Lock()
	defer r.tsGenMu.Unlock()
	if r.tsGenCfg == nil {
		return nil
	}
	c := *r.tsGenCfg
	return &c
}

// GenerateTSClient generates the TypeScript client using the configured TSClientGenConfig.
// No-op if no config is set. Errors are returned to the caller, and also passed to cfg.OnError if provided.
func (r *Router) GenerateTSClient() error {
	cfg := r.tsClientGenConfig()
	if cfg == nil {
		return nil
	}
	return r.generateTSClient(*cfg)
}

func (r *Router) generateTSClient(cfg TSClientGenConfig) error {
	outDir := strings.TrimSpace(cfg.Dir)
	if outDir == "" {
		return nil
	}

	if !filepath.IsAbs(outDir) {
		if cwd, err := os.Getwd(); err == nil {
			outDir = filepath.Join(cwd, outDir)
		}
	}
	outDir = filepath.Clean(outDir)

	opts := cfg.Options.withDefaults()

	if err := r.GenTSDir(outDir, opts); err != nil {
		if cfg.OnError != nil {
			cfg.OnError(err)
		}
		return err
	}
	return nil
}
