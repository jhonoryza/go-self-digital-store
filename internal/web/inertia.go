package web

// Setup gonertia v3 + resolver aset shell Go. Dua mode:
//   - hot reload (web/hot ada, saat `bun run dev`): shell Go merujuk dev server;
//   - bundled (web/dist/.vite/manifest.json): entri "index.html" dibaca langsung
//     (js + css) — tahan perubahan bentuk manifest antar versi Vite.
// Di produksi utama shell statis CF Pages mem-boot app yang sama via fetch page
// JSON (lihat web/src/api.ts); shell Go dipakai dev/single-origin & fallback.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/config"
)

const (
	viteManifest = "web/dist/.vite/manifest.json"
	viteHotFile  = "web/hot"
)

// rootTemplate shell HTML yang dirender Go; aset diisi lewat template data
// (appJS/appCSS) agar tidak terikat bentuk manifest.
const rootTemplate = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title inertia>{{ .appName }}</title>
{{ if .hmr }}<script type="module" src="{{ .hmrURL }}/@vite/client"></script>{{ end }}
<link rel="stylesheet" href="{{ .appCSS }}"/>
<script type="module" src="{{ .appJS }}"></script>
{{ .inertiaHead }}
</head>
<body>{{ .inertia }}</body>
</html>
`

// shellAssets URL entry JS + CSS untuk shell.
type shellAssets struct{ JS, CSS, HMR string }

// resolveAssets baca manifest/hot file; fallback kosong bila belum di-build.
func resolveAssets() shellAssets {
	if raw, err := os.ReadFile(viteHotFile); err == nil {
		hmr := strings.TrimSpace(string(raw))
		if hmr == "" {
			hmr = "//localhost:5173"
		}
		if strings.HasPrefix(hmr, "http://") || strings.HasPrefix(hmr, "https://") {
			hmr = "//" + hmr[strings.Index(hmr, "//")+2:]
		}
		return shellAssets{JS: hmr + "/src/app.tsx", CSS: hmr + "/src/css/app.css", HMR: hmr}
	}

	b, err := os.ReadFile(viteManifest)
	if err != nil {
		log.Printf("peringatan: manifest vite belum ada (%s) — shell tanpa aset", viteManifest)
		return shellAssets{}
	}
	var m map[string]struct {
		File    string   `json:"file"`
		IsEntry bool     `json:"isEntry"`
		CSS     []string `json:"css"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		log.Printf("peringatan: manifest rusak: %v", err)
		return shellAssets{}
	}
	for _, e := range m {
		if !e.IsEntry {
			continue
		}
		a := shellAssets{JS: "/" + e.File}
		if len(e.CSS) > 0 {
			a.CSS = "/" + e.CSS[0]
		}
		return a
	}
	return shellAssets{}
}

// NewInertia bangun instance gonertia; kembalikan juga versi aset
// (dipakai header X-Inertia-Version).
func NewInertia(cfg config.Config) (*inertia.Inertia, string) {
	version := assetVersion()
	i, err := inertia.New(rootTemplate, inertia.WithVersion(version))
	if err != nil {
		log.Fatalf("inertia: %v", err)
	}
	i.ShareTemplateData("appName", cfg.AppName)

	assets := resolveAssets()
	i.ShareTemplateData("appJS", assets.JS)
	i.ShareTemplateData("appCSS", assets.CSS)
	i.ShareTemplateData("hmr", assets.HMR != "")
	i.ShareTemplateData("hmrURL", assets.HMR)

	return i, version
}

// assetVersion "dev" saat hot reload; hash manifest saat bundled.
func assetVersion() string {
	if _, err := os.ReadFile(viteHotFile); err == nil {
		return "dev"
	}
	b, err := os.ReadFile(viteManifest)
	if err != nil {
		return "1"
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}
