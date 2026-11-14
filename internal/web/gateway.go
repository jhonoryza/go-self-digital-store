package web

// Integrasi paygateway: konfigurasi provider di settings runtime (dikelola
// konsol admin, M6), pembuatan tagihan saat checkout, dan provider aktif untuk
// webhook. Pola diadaptasi dari linkaku.

import (
	"database/sql"
	"encoding/json"

	pay "github.com/jhonoryza/paygateway"
)

// providerCfg konfigurasi provider tersimpan sebagai JSON di settings.
func providerCfg(db *sql.DB, name string) pay.Config {
	var raw string
	db.QueryRow(`SELECT value FROM settings WHERE key=?`, "payment_cfg."+name).Scan(&raw)
	cfg := pay.Config{}
	json.Unmarshal([]byte(raw), &cfg)
	return cfg
}

// activeProvider provider aktif + cfg; fallback Mock (simulasi) bila belum diset.
func activeProvider(db *sql.DB) (pay.Provider, pay.Config, string) {
	name := getSetting(db, "payment_provider")
	if p, ok := pay.Get(name); ok {
		return p, providerCfg(db, name), name
	}
	return pay.Mock{}, pay.Config{}, "mock"
}

// isSimulated true bila provider aktif adalah mode simulasi.
func isSimulated(p pay.Provider) bool { return pay.IsSimulated(p) }

// createCharge buat tagihan gateway untuk order; kembalikan PayURL (kosong
// saat simulasi atau VA tanpa halaman). Error -> order tetap pending, pembeli
// bisa coba lagi.
func createCharge(db *sql.DB, orderNo string, amount int, title, email, callbackURL, returnURL string) (payURL, provider string, err error) {
	p, cfg, name := activeProvider(db)
	if isSimulated(p) {
		return "", "mock", nil
	}
	res, err := p.CreateCharge(pay.Charge{
		OrderID:     orderNo,
		Amount:      amount,
		Title:       title,
		BuyerEmail:  email,
		CallbackURL: callbackURL,
		ReturnURL:   returnURL,
	}, cfg)
	if err != nil {
		return "", name, err
	}
	return res.PayURL, name, nil
}
