package httprpc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// TSClientGenConfig configures TypeScript client generation.
type TSClientGenConfig struct {
	// Dir is the output directory for generated TypeScript.
	// If empty, generation is disabled.
	Dir string

	// Options are passed through to GenTSDir.
	Options TSGenOptions

	// OnError is called if generation fails during Router.Handler().
	// If nil, errors are ignored.
	OnError func(error)
}

const tsClientChecksumFileName = ".httprpc-checksum"

// SetTSClientGenConfig sets the configuration for TypeScript client generation.
func (r *Router) SetTSClientGenConfig(cfg *TSClientGenConfig) {
	r.tsGenMu.Lock()
	defer r.tsGenMu.Unlock()

	if cfg == nil || strings.TrimSpace(cfg.Dir) == "" {
		r.tsGenCfg = nil
		r.tsGenLastDir = ""
		r.tsGenLastHash = ""
		return
	}
	c := *cfg
	c.Options = c.Options.withDefaults()
	r.tsGenCfg = &c
	r.tsGenLastDir = ""
	r.tsGenLastHash = ""
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

func (r *Router) maybeGenTS(cfg TSClientGenConfig) {
	outDir := strings.TrimSpace(cfg.Dir)
	if outDir == "" {
		return
	}

	if !filepath.IsAbs(outDir) {
		if cwd, err := os.Getwd(); err == nil {
			outDir = filepath.Join(cwd, outDir)
		}
	}
	outDir = filepath.Clean(outDir)

	opts := cfg.Options.withDefaults()
	sum := tsClientChecksum(r.Metas, opts)

	r.tsGenMu.Lock()
	if r.tsGenLastDir == outDir && r.tsGenLastHash == sum {
		r.tsGenMu.Unlock()
		return
	}
	r.tsGenMu.Unlock()

	if existing, err := readTSClientChecksum(outDir); err == nil && existing == sum {
		r.tsGenMu.Lock()
		r.tsGenLastDir = outDir
		r.tsGenLastHash = sum
		r.tsGenMu.Unlock()
		return
	}

	if err := r.GenTSDir(outDir, opts); err != nil {
		if cfg.OnError != nil {
			cfg.OnError(err)
		}
		return
	}

	r.tsGenMu.Lock()
	r.tsGenLastDir = outDir
	r.tsGenLastHash = sum
	r.tsGenMu.Unlock()
}

func readTSClientChecksum(dir string) (string, error) {
	if b, err := os.ReadFile(filepath.Clean(filepath.Join(dir, tsClientChecksumFileName))); err == nil {
		sum := strings.TrimSpace(string(b))
		if sum == "" {
			return "", fmt.Errorf("checksum empty")
		}
		return sum, nil
	}

	b, err := os.ReadFile(filepath.Clean(filepath.Join(dir, "base.ts")))
	if err != nil {
		return "", fmt.Errorf("read base.ts: %w", err)
	}
	s := string(b)
	const marker = "export const __httprpc_checksum = "
	i := strings.Index(s, marker)
	if i < 0 {
		return "", fmt.Errorf("checksum marker not found")
	}
	s = strings.TrimLeft(s[i+len(marker):], " \t")
	if !strings.HasPrefix(s, "\"") {
		return "", fmt.Errorf("checksum marker is not a string literal")
	}
	j := strings.IndexByte(s[1:], '"')
	if j < 0 {
		return "", fmt.Errorf("checksum marker is not a complete string literal")
	}
	lit := s[:j+2]
	v, err := strconv.Unquote(lit)
	if err != nil {
		return "", fmt.Errorf("unquote checksum: %w", err)
	}
	if v == "" {
		return "", fmt.Errorf("checksum empty")
	}
	return v, nil
}

const extraLines = 4

func tsClientChecksum(metas []*EndpointMeta, opts TSGenOptions) string {
	lines := make([]string, 0, len(metas)+extraLines)
	lines = append(lines, "package="+opts.PackageName, "client="+opts.ClientName, fmt.Sprintf("skip=%d", opts.SkipPathSegments))

	for _, m := range metas {
		if m == nil {
			continue
		}
		lines = append(lines, strings.Join([]string{
			strings.ToUpper(m.Method),
			m.Path,
			typeKey(m.Req),
			typeKey(m.Res),
			strings.Join(m.Consumes, ","),
			strings.Join(m.Produces, ","),
		}, "|"))
	}
	sort.Strings(lines[3:])

	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

func typeKey(t reflect.Type) string {
	if t == nil {
		return ""
	}
	return t.PkgPath() + ":" + t.String()
}
