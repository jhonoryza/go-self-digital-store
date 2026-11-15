package web

// Setup gonertia v3 + resolver aset shell Go. Dua mode:
//   - hot reload (web/hot ada, saat `bun run dev`): shell Go merujuk dev server;
//   - bundled (web/dist/.vite/manifest.json): entri "index.html" dibaca langsung
//     (js + css) — tahan perubahan bentuk manifest antar versi Vite.
// Di produksi utama shell statis CF Pages mem-boot app yang sama via fetch page
// JSON (lihat web/src/api.ts); shell Go dipakai dev/single-origin & fallback.

import (
	"crypto/md5"
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
<html lang="id" class="dark" data-theme="{{ .theme }}">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title inertia>{{ .appName }}</title>
<script>
// tema diputuskan sebelum paint agar tak berkedip; default gelap,
// pilihan user (localStorage) menang.
(function () {
  try {
    var t = localStorage.getItem("theme");
    if (t === "light") document.documentElement.classList.remove("dark");
    else if (t === "dark") document.documentElement.classList.add("dark");
  } catch (e) {}
})();
</script>
{{ if .hmr }}<script type="module" src="{{ .hmrURL }}/@vite/client"></script>{{ end }}
<link rel="stylesheet" href="{{ .appCSS }}"/>
<script type="module" src="{{ .appJS }}"></script>
{{ if .ogTitle }}<meta property="og:title" content="{{ .ogTitle }}"/>
<meta property="og:description" content="{{ .ogDesc }}"/>
<meta property="og:type" content="{{ .ogType }}"/>
<meta property="og:site_name" content="{{ .appName }}"/>
{{ if .ogURL }}<meta property="og:url" content="{{ .ogURL }}"/>{{ end }}
{{ if .ogImage }}<meta property="og:image" content="{{ .ogImage }}"/>{{ end }}{{ end }}
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
	version := inertiaVersion(assetVersion())
	i, err := inertia.New(rootTemplate, inertia.WithVersion(assetVersion()))
	if err != nil {
		log.Fatalf("inertia: %v", err)
	}
	i.ShareTemplateData("appName", cfg.AppName)
	i.ShareTemplateData("theme", "") // ditimpa per-request di Page()

	assets := resolveAssets()
	i.ShareTemplateData("appJS", assets.JS)
	i.ShareTemplateData("appCSS", assets.CSS)
	i.ShareTemplateData("hmr", assets.HMR != "")
	i.ShareTemplateData("hmrURL", assets.HMR)

	return i, version
}

// inertiaVersion cermin cara gonertia menyimpan versi (md5 hex argumen) —
// middleware pembanding X-Inertia-Version wajib memakai nilai yang sama
// persis dengan yang dikirim di JSON halaman, bukan versi mentah.
func inertiaVersion(v string) string {
	sum := md5.Sum([]byte(v))
	return hex.EncodeToString(sum[:])
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
