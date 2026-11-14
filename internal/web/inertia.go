package web

// Setup gonertia v3 + integrasi Vite. Dua mode shell:
//   - hot reload (web/hot ada, saat `bun run dev`): shell Go merujuk dev server
//   - bundled (web/dist/.vite/manifest.json): shell Go merujuk aset ber-hash —
//     dipakai dev tanpa vite dan fallback produksi; di produksi utama shell
//     statis CF Pages mem-boot app yang sama via fetch page JSON (lihat web/src/app.tsx).

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/config"
)

const (
	viteManifest = "web/dist/.vite/manifest.json"
	viteHotFile  = "web/hot"
	appEntry     = "src/app.tsx"
	cssEntry     = "src/css/app.css"
)

// rootTemplate shell HTML yang dirender Go; aset di-resolve lewat manifest/hot file.
const rootTemplate = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title inertia>{{ .appName }}</title>
{{ if .hmr }}{{ viteRefresh }}{{ viteReactRefresh }}{{ end }}
<link rel="stylesheet" href="{{ vite "` + cssEntry + `"}}"/>
<script type="module" src="{{ vite "` + appEntry + `"}}"></script>
{{ .inertiaHead }}
</head>
<body>{{ .inertia }}</body>
</html>
`

// NewInertia bangun instance gonertia + vite; kembalikan juga versi aset
// (dipakai header X-Inertia-Version).
func NewInertia(cfg config.Config) (*inertia.Inertia, string) {
	version := assetVersion()
	i, err := inertia.New(rootTemplate, inertia.WithVersion(version))
	if err != nil {
		log.Fatalf("inertia: %v", err)
	}
	i.ShareTemplateData("appName", cfg.AppName)

	_, err = inertia.NewVite(i,
		inertia.WithHotFile(viteHotFile),
		inertia.WithBuildManifest(viteManifest),
		inertia.WithBuildDir("/"),
	)
	if err != nil {
		log.Fatalf("vite: %v", err)
	}
	return i, version
}

// assetVersion "dev" saat hot reload; hash manifest saat bundled.
func assetVersion() string {
	if _, err := os.Stat(viteHotFile); err == nil {
		return "dev"
	}
	b, err := os.ReadFile(viteManifest)
	if err != nil {
		return "1"
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}
