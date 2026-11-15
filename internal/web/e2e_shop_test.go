package web

// Blok test dipecah per domain dari e2e_test.go (helper tetap di sana).

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---- dashboard, pesanan, invoice ----

func TestE2EDashboardOrdersInvoice(t *testing.T) {
	s, ts, d := testServer(t)
	fileID := seedProduct(t, d, "berkas-lengkap", 120000, "file")
	if err := writeFileForTest(t, s.filesDir(), "berkas-lengkap.pdf", "DATA"); err != nil {
		t.Fatal(err)
	}
	d.Exec(`UPDATE products SET file_path='berkas-lengkap.pdf', file_name='Berkas Lengkap.pdf' WHERE id=?`, fileID)

	// register + verifikasi email (checkout kini wajib login terverifikasi)
	client := newClient()
	doPost(t, client, ts.URL+"/register", `{"name":"Pembeli","email":"pembeli-lengkap@uji.id","password":"rahasia123"}`)
	verifyUser(t, s, d, client, ts, "pembeli-lengkap@uji.id")

	// dua checkout sebagai user terverifikasi
	ordGuest, tokGuest := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"pembeli-lengkap@uji.id"}`, fileID))

	ordUser, _ := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"pembeli-lengkap@uji.id"}`, fileID))
	var uid sql.NullInt64
	d.QueryRow(`SELECT user_id FROM orders WHERE order_no=?`, ordUser).Scan(&uid)
	if !uid.Valid {
		t.Error("order checkout login tidak terikat user_id")
	}

	// daftar pesanan memuat keduanya
	page := doPage(t, client, ts.URL+"/pesanan")
	orders := prop(t, page)["orders"].([]any)
	if len(orders) != 2 {
		t.Fatalf("orders = %d, mau 2", len(orders))
	}

	// invoice milik sendiri (order guest pun) bisa dibuka
	page = doPage(t, client, ts.URL+"/pesanan/"+ordGuest)
	o := prop(t, page)["order"].(map[string]any)
	if o["order_no"] != ordGuest || o["status"] != "pending" {
		t.Errorf("invoice = %v status %v", o["order_no"], o["status"])
	}
	if len(o["items"].([]any)) != 1 {
		t.Errorf("item invoice = %v", o["items"])
	}

	// order milik orang lain → 404 (user lain harus verified dulu juga)
	doPost(t, newClient(), ts.URL+"/register", `{"name":"Lain","email":"orang-lain@uji.id","password":"rahasia123"}`)
	other := loginCookie(t, ts, "orang-lain@uji.id", "rahasia123")
	verifyUser(t, s, d, other, ts, "orang-lain@uji.id")
	res := doGetRes(t, other, ts.URL+"/pesanan/"+ordGuest, map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("invoice orang lain = %d, mau 404", res.StatusCode)
	}

	// pelunasan simulasi → dashboard stats benar
	res = paySimulasi(t, client, ts, ordGuest, tokGuest)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("paySimulasi = %d", res.StatusCode)
	}
	page = doPage(t, client, ts.URL+"/dashboard")
	stats := prop(t, page)["stats"].(map[string]any)
	if stats["paid_count"].(float64) != 1 || stats["pending_count"].(float64) != 1 {
		t.Errorf("stats hitung = %v", stats)
	}
	if stats["total_spend"].(float64) != 120000 {
		t.Errorf("total_spend = %v", stats["total_spend"])
	}
	if len(stats["recent"].([]any)) != 2 {
		t.Errorf("recent = %d", len(stats["recent"].([]any)))
	}
	// invoice kini status paid
	page = doPage(t, client, ts.URL+"/pesanan/"+ordGuest)
	if prop(t, page)["order"].(map[string]any)["status"] != "paid" {
		t.Error("invoice belum paid")
	}
}

// ---- validasi checkout + kupon ----

func TestE2ECheckoutValidation(t *testing.T) {
	_, ts, d := testServer(t)
	pid := seedProduct(t, d, "produk-checkout", 100000, "key")
	seedKeys(t, d, pid, 1)
	deadID := seedProduct(t, d, "produk-mati", 50000, "file")
	d.Exec(`UPDATE products SET active=0 WHERE id=?`, deadID)

	seedUser(t, d, "validator@uji.id", "rahasia123", "user")
	guest := loginCookie(t, ts, "validator@uji.id", "rahasia123")
	vcsrf := csrfOf(t, guest, ts, "/")

	// keranjang kosong → redirect /keranjang dengan flash err
	res := doPostCSRF(t, guest, ts.URL+"/checkout", `{}`, vcsrf)
	if res.StatusCode != http.StatusFound || !strings.HasPrefix(res.Header.Get("Location"), "/keranjang?") ||
		!strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("keranjang kosong = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// email tidak valid → render ulang dengan errors
	res = doPostCSRF(t, guest, ts.URL+"/checkout", fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"ngawur"}`, pid), vcsrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("email salah = %d", res.StatusCode)
	}
	if !errHas(postErrors(t, res), "email") {
		t.Error("errors.email tidak ada")
	}

	// qty di luar 1..10 ditolak
	for _, qty := range []int{0, 11} {
		res = doPostCSRF(t, guest, ts.URL+"/checkout",
			fmt.Sprintf(`{"items":[{"id":%d,"qty":%d}],"email":"p@uji.id"}`, pid, qty), vcsrf)
		if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
			t.Errorf("qty %d = %d %s", qty, res.StatusCode, res.Header.Get("Location"))
		}
	}

	// produk nonaktif ditolak
	res = doPostCSRF(t, guest, ts.URL+"/checkout",
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"p@uji.id"}`, deadID), vcsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("produk mati = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// kupon valid: diskon 10% (kode lowercase dinormalkan)
	seedCoupon(t, d, "HEMAT10", 10)
	orderNo, _ := checkoutOrder(t, guest, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"p@uji.id","coupon":"hemat10"}`, pid))
	var subtotal, discount, total int
	var coupon string
	d.QueryRow(`SELECT subtotal, discount, total, coupon_code FROM orders WHERE order_no=?`, orderNo).
		Scan(&subtotal, &discount, &total, &coupon)
	if subtotal != 100000 || discount != 10000 || total != 90000 || coupon != "HEMAT10" {
		t.Errorf("kupon: subtotal=%d discount=%d total=%d code=%q", subtotal, discount, total, coupon)
	}

	// kupon tak dikenal → harga penuh
	orderNo, _ = checkoutOrder(t, guest, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"p@uji.id","coupon":"NGASAL"}`, pid))
	d.QueryRow(`SELECT subtotal, discount, total FROM orders WHERE order_no=?`, orderNo).Scan(&subtotal, &discount, &total)
	if discount != 0 || total != 100000 {
		t.Errorf("kupon ngasal: discount=%d total=%d", discount, total)
	}
}

// ---- pembayaran simulasi (mock) ----

func TestE2ESimulatedPayment(t *testing.T) {
	s, ts, d := testServer(t)
	keyID := seedProduct(t, d, "kunci-sim", 25000, "key")
	seedKeys(t, d, keyID, 3)
	noKeyID := seedProduct(t, d, "kunci-habis", 25000, "key") // tanpa stok

	seedUser(t, d, "sim@uji.id", "rahasia123", "user")
	client := loginCookie(t, ts, "sim@uji.id", "rahasia123")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":2}],"email":"sim@uji.id"}`, keyID))

	// halaman bayar: token benar → props simulasi + order pending
	page := doPage(t, client, fmt.Sprintf("%s/pay/%s?t=%s", ts.URL, orderNo, token))
	pp := prop(t, page)
	if pp["simulasi"] != true {
		t.Errorf("simulasi = %v", pp["simulasi"])
	}
	if o := pp["order"].(map[string]any); o["status"] != "pending" || o["total"].(float64) != 50000 {
		t.Errorf("order pay page = %v", o)
	}

	// token salah / kosong / order tak dikenal → 404
	res := doGetRes(t, client, fmt.Sprintf("%s/pay/%s?t=salah", ts.URL, orderNo), nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("token salah = %d", res.StatusCode)
	}
	res = doGetRes(t, client, fmt.Sprintf("%s/pay/%s", ts.URL, orderNo), nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("token kosong = %d", res.StatusCode)
	}
	res = doGetRes(t, client, ts.URL+"/pay/ORD-TAKADA?t=x", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("order tak ada = %d", res.StatusCode)
	}

	// payNow token salah → 404
	res = doPost(t, client, ts.URL+"/pay/"+orderNo, `{"t":"salah"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("payNow token salah = %d", res.StatusCode)
	}

	// payNow benar → lunas, kunci 2 unit, email tercatat
	res = paySimulasi(t, client, ts, orderNo, token)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("payNow = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	var st string
	d.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo).Scan(&st)
	if st != "paid" {
		t.Fatalf("status = %q", st)
	}
	var assigned int
	d.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&assigned)
	if assigned != 2 {
		t.Errorf("kunci = %d, mau 2", assigned)
	}
	waitMailLog(t, d, "sim@uji.id", 1)

	// halaman bayar kini menampilkan kunci ter-assign
	page = doPage(t, client, fmt.Sprintf("%s/pay/%s?t=%s", ts.URL, orderNo, token))
	o := prop(t, page)["order"].(map[string]any)
	keys := o["items"].([]any)[0].(map[string]any)["license_keys"].([]any)
	if len(keys) != 2 {
		t.Errorf("kunci di halaman = %d", len(keys))
	}

	// payNow ulang → idempoten, kunci tetap 2
	res = paySimulasi(t, client, ts, orderNo, token)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("payNow ulang = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	d.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&assigned)
	if assigned != 2 {
		t.Errorf("kunci setelah ulang = %d", assigned)
	}

	// order kunci tanpa stok: checkout DITOLAK (tidak boleh jual kunci
	// yang tidak ada; kunci menyusul hanya terjadi pada balapan order)
	res = doPostCSRF(t, client, ts.URL+"/checkout",
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"habis@uji.id"}`, noKeyID), csrfOf(t, client, ts, "/keranjang"))
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=Stok+kunci") {
		t.Errorf("checkout tanpa stok = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// webhook saat mode simulasi → 404 (bukan jalur callback)
	res = doPost(t, &http.Client{}, ts.URL+"/callback/payment", `{"order_id":"x","status":"paid","amount":1}`)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("callback simulasi = %d, mau 404", res.StatusCode)
	}

	// payNow saat provider non-simulasi aktif → 404 (order tak dikenal)
	setSetting(s.DB, "payment_provider", "tripay")
	setSetting(s.DB, "payment_cfg.tripay", `{"private_key":"k"}`)
	res = doPost(t, client, ts.URL+"/pay/ORD-TAKADA-LAGI", `{"t":"x"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("payNow non-simulasi = %d, mau 404", res.StatusCode)
	}
}

// ---- webhook tripay: jalur negatif ----

func TestE2ETripayWebhookNegatives(t *testing.T) {
	s, ts, d := testServer(t)
	fileID := seedProduct(t, d, "berkas-webhook", 200000, "file")
	seedUser(t, d, "hook@uji.id", "rahasia123", "user")
	hookClient := loginCookie(t, ts, "hook@uji.id", "rahasia123")
	orderNo, _ := checkoutOrder(t, hookClient, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"hook@uji.id"}`, fileID))

	setSetting(s.DB, "payment_provider", "tripay")
	setSetting(s.DB, "payment_cfg.tripay", `{"private_key":"kunci-tripay"}`)

	sign := func(body string) http.Header {
		h := http.Header{}
		h.Set("X-Callback-Signature", hmacHexOf("kunci-tripay", body))
		h.Set("Content-Type", "application/json")
		return h
	}
	sendHook := func(body string, hdr http.Header) (int, string) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/callback/payment", strings.NewReader(body))
		req.Header = hdr
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}

	// status bukan PAID → diabaikan, order tetap pending
	body := fmt.Sprintf(`{"merchant_ref":%q,"status":"EXPIRED","amount":200000,"reference":"r1"}`, orderNo)
	code, respBody := sendHook(body, sign(body))
	if code != http.StatusOK || !strings.Contains(respBody, `"ignored":true`) {
		t.Errorf("EXPIRED = %d %s", code, respBody)
	}
	var st string
	d.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo).Scan(&st)
	if st != "pending" {
		t.Errorf("status setelah EXPIRED = %q", st)
	}

	// nominal tidak cocok → tidak lunas walau status PAID
	body = fmt.Sprintf(`{"merchant_ref":%q,"status":"PAID","amount":199999,"reference":"r2"}`, orderNo)
	code, respBody = sendHook(body, sign(body))
	if code != http.StatusOK || !strings.Contains(respBody, `"ok":true`) {
		t.Errorf("nominal beda = %d %s", code, respBody)
	}
	d.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo).Scan(&st)
	if st != "pending" {
		t.Errorf("status setelah nominal beda = %q", st)
	}

	// order tak dikenal → ok tanpa efek
	body = `{"merchant_ref":"ORD-TAKADA","status":"PAID","amount":1000,"reference":"r3"}`
	code, _ = sendHook(body, sign(body))
	if code != http.StatusOK {
		t.Errorf("order tak dikenal = %d", code)
	}

	// signature salah → 401
	body = fmt.Sprintf(`{"merchant_ref":%q,"status":"PAID","amount":200000,"reference":"r4"}`, orderNo)
	bad := sign(body)
	bad.Set("X-Callback-Signature", "deadbeef")
	code, _ = sendHook(body, bad)
	if code != http.StatusUnauthorized {
		t.Errorf("signature salah = %d, mau 401", code)
	}

	// fail-closed: secret kosong → 401 walau signature "benar"
	setSetting(s.DB, "payment_cfg.tripay", `{}`)
	code, _ = sendHook(body, sign(body))
	if code != http.StatusUnauthorized {
		t.Errorf("cfg kosong = %d, mau 401", code)
	}

	// method salah → 404 (catch-all 404 menang atas mismatch method; tetap
	// tidak membocorkan keberadaan endpoint)
	res := doGetRes(t, newClient(), ts.URL+"/callback/payment", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("GET callback = %d, mau 404", res.StatusCode)
	}
}

// ---- unduhan /dl ----

func TestE2EDownloadGuards(t *testing.T) {
	s, ts, d := testServer(t)
	fileID := seedProduct(t, d, "berkas-dl", 50000, "file")
	keyID := seedProduct(t, d, "kunci-dl", 50000, "key")
	seedKeys(t, d, keyID, 1)
	if err := writeFileForTest(t, s.filesDir(), "berkas-dl.zip", "ZIP-DATA-RAHASIA"); err != nil {
		t.Fatal(err)
	}
	d.Exec(`UPDATE products SET file_path='berkas-dl.zip', file_name='Berkas DL.zip' WHERE id=?`, fileID)

	seedUser(t, d, "dl@uji.id", "rahasia123", "user")
	client := loginCookie(t, ts, "dl@uji.id", "rahasia123")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1},{"id":%d,"qty":1}],"email":"dl@uji.id"}`, fileID, keyID))

	var fileItem, keyItem int64
	d.QueryRow(`SELECT oi.id FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE p.type='file'`).Scan(&fileItem)
	d.QueryRow(`SELECT oi.id FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE p.type='key'`).Scan(&keyItem)

	dl := func(item int64, tok string) *http.Response {
		return doGetRes(t, &http.Client{}, fmt.Sprintf("%s/dl/%s/%d?t=%s", ts.URL, orderNo, item, tok), nil)
	}

	// sebelum lunas → 403
	if res := dl(fileItem, token); res.StatusCode != http.StatusForbidden {
		t.Errorf("belum lunas = %d, mau 403", res.StatusCode)
	}

	// lunas via simulasi
	if res := paySimulasi(t, client, ts, orderNo, token); res.StatusCode != http.StatusFound {
		t.Fatalf("payNow = %d", res.StatusCode)
	}

	// token salah / kosong → 403
	if res := dl(fileItem, "salah"); res.StatusCode != http.StatusForbidden {
		t.Errorf("token salah = %d", res.StatusCode)
	}
	res := doGetRes(t, &http.Client{}, fmt.Sprintf("%s/dl/%s/%d", ts.URL, orderNo, fileItem), nil)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("token kosong = %d", res.StatusCode)
	}

	// item bukan type=file / item tak ada → 404
	if res := dl(keyItem, token); res.StatusCode != http.StatusNotFound {
		t.Errorf("item kunci = %d, mau 404", res.StatusCode)
	}
	if res := dl(99999, token); res.StatusCode != http.StatusNotFound {
		t.Errorf("item ngawur = %d, mau 404", res.StatusCode)
	}

	// unduhan sah → 200 + nama berkas + isi
	res = dl(fileItem, token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("unduhan = %d", res.StatusCode)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, "Berkas DL.zip") {
		t.Errorf("disposition = %q", cd)
	}
	b, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(b), "ZIP-DATA-RAHASIA") {
		t.Errorf("isi = %q", truncate(string(b), 100))
	}
}

// ---- tiket: siklus penuh ----

func TestE2ETicketsLifecycle(t *testing.T) {
	_, ts, d := testServer(t)
	uid := seedUser(t, d, "op@uji.id", "rahasia123", "user")
	seedUser(t, d, "lain@uji.id", "rahasia123", "user")
	seedUser(t, d, "admin@uji.id", "adminpass1", "admin")
	pid := seedProduct(t, d, "berkas-tiket", 10000, "file")
	seedOrder(t, d, "ORD-TIKET", "op@uji.id", pid, 10000, 1)
	seedOrder(t, d, "ORD-ORANG", "lain@uji.id", pid, 10000, 1)

	user := loginCookie(t, ts, "op@uji.id", "rahasia123")
	csrf := csrfOf(t, user, ts, "/tiket")

	// validasi: subjek & pesan wajib
	res := doPostCSRF(t, user, ts.URL+"/tiket", `{"subject":"","body":""}`, csrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("tiket kosong = %d", res.StatusCode)
	}
	errs := postErrors(t, res)
	if !errHas(errs, "subject") || !errHas(errs, "body") {
		t.Errorf("errors tiket = %v", errs)
	}

	// order milik orang lain ditolak
	res = doPostCSRF(t, user, ts.URL+"/tiket", `{"subject":"S","body":"B","order_no":"ORD-ORANG"}`, csrf)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "order_no") {
		t.Errorf("order orang lain = %d", res.StatusCode)
	}

	// buat tiket terkait order sendiri
	res = doPostCSRF(t, user, ts.URL+"/tiket",
		`{"subject":"Berkas rusak","body":"Tautan gagal dibuka","order_no":"ORD-TIKET"}`, csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("buat tiket = %d", res.StatusCode)
	}
	var tid int64
	var orderNo, status string
	d.QueryRow(`SELECT id, order_no, status FROM tickets WHERE user_id=?`, uid).Scan(&tid, &orderNo, &status)
	if orderNo != "ORD-TIKET" || status != "open" {
		t.Errorf("tiket = id %d order %q status %q", tid, orderNo, status)
	}

	// user lain tak bisa membuka tiket itu
	other := loginCookie(t, ts, "lain@uji.id", "rahasia123")
	res = doGetRes(t, other, fmt.Sprintf("%s/tiket/%d", ts.URL, tid), map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("tiket orang lain = %d, mau 404", res.StatusCode)
	}

	// admin balas → answered
	admin := loginCookie(t, ts, "admin@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")
	res = doPostCSRF(t, admin, fmt.Sprintf("%s/admin/tiket/%d", ts.URL, tid), `{"body":"Sudah diperbaiki"}`, acsrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("balas admin = %d", res.StatusCode)
	}
	d.QueryRow(`SELECT status FROM tickets WHERE id=?`, tid).Scan(&status)
	if status != "answered" {
		t.Errorf("status = %q, mau answered", status)
	}

	// balasan user → kembali open
	res = doPostCSRF(t, user, fmt.Sprintf("%s/tiket/%d", ts.URL, tid), `{"body":"Masih gagal"}`, csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("balas user = %d", res.StatusCode)
	}
	d.QueryRow(`SELECT status FROM tickets WHERE id=?`, tid).Scan(&status)
	if status != "open" {
		t.Errorf("status = %q, mau open", status)
	}
	var msgs int
	d.QueryRow(`SELECT COUNT(*) FROM ticket_messages WHERE ticket_id=?`, tid).Scan(&msgs)
	if msgs != 3 {
		t.Errorf("pesan = %d, mau 3", msgs)
	}

	// admin tutup → closed
	res = doPostCSRF(t, admin, fmt.Sprintf("%s/admin/tiket/%d/close", ts.URL, tid), `{}`, acsrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("tutup tiket = %d", res.StatusCode)
	}
	d.QueryRow(`SELECT status FROM tickets WHERE id=?`, tid).Scan(&status)
	if status != "closed" {
		t.Errorf("status = %q, mau closed", status)
	}

	// balas tiket tertutup → flash err, tanpa pesan baru
	res = doPostCSRF(t, user, fmt.Sprintf("%s/tiket/%d", ts.URL, tid), `{"body":"Lagi"}`, csrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("balas closed = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	d.QueryRow(`SELECT COUNT(*) FROM ticket_messages WHERE ticket_id=?`, tid).Scan(&msgs)
	if msgs != 3 {
		t.Errorf("pesan setelah closed = %d, mau tetap 3", msgs)
	}

	// daftar admin memuat email pemilik + detail pesan
	page := doPage(t, admin, ts.URL+"/admin/tiket")
	found := false
	for _, it := range prop(t, page)["tickets"].([]any) {
		if m := it.(map[string]any); int64(m["id"].(float64)) == tid && m["email"] == "op@uji.id" {
			found = true
		}
	}
	if !found {
		t.Error("tiket tidak muncul di daftar admin")
	}
	page = doPage(t, admin, fmt.Sprintf("%s/admin/tiket/%d", ts.URL, tid))
	tk := prop(t, page)["ticket"].(map[string]any)
	if len(tk["messages"].([]any)) != 3 {
		t.Errorf("pesan detail admin = %d", len(tk["messages"].([]any)))
	}

	// balasan admin tanpa pesan → errors
	res = doPostCSRF(t, admin, fmt.Sprintf("%s/admin/tiket/%d", ts.URL, tid), `{"body":""}`, acsrf)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "body") {
		t.Errorf("balas kosong = %d", res.StatusCode)
	}
}

// ---- alur lintas fitur: beli → tiket → invoice (satu sesi user) ----

func TestE2EBuyThenTicketThenInvoice(t *testing.T) {
	s, ts, d := testServer(t)
	keyID := seedProduct(t, d, "kunci-lintas", 75000, "key")
	seedKeys(t, d, keyID, 1)

	client := newClient()
	doPost(t, client, ts.URL+"/register", `{"name":"Rina","email":"rina@uji.id","password":"rahasia123"}`)
	verifyUser(t, s, d, client, ts, "rina@uji.id")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"rina@uji.id"}`, keyID))

	// tiket merujuk order yang baru dibuat
	csrf := csrfOf(t, client, ts, "/tiket")
	res := doPostCSRF(t, client, ts.URL+"/tiket",
		fmt.Sprintf(`{"subject":"Tanya kunci","body":"Kapan dikirim?","order_no":%q}`, orderNo), csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("tiket dgn order = %d", res.StatusCode)
	}

	// bayar → invoice menampilkan kunci; dashboard terhitung
	if res := paySimulasi(t, client, ts, orderNo, token); res.StatusCode != http.StatusFound {
		t.Fatalf("payNow = %d", res.StatusCode)
	}
	page := doPage(t, client, ts.URL+"/pesanan/"+orderNo)
	o := prop(t, page)["order"].(map[string]any)
	item := o["items"].([]any)[0].(map[string]any)
	ks := item["license_keys"].([]any)
	if len(ks) != 1 {
		t.Fatalf("kunci invoice = %v", ks)
	}
	var keyVal string
	d.QueryRow(`SELECT key FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&keyVal)
	if ks[0] != keyVal {
		t.Errorf("kunci invoice %v ≠ DB %q", ks[0], keyVal)
	}
	page = doPage(t, client, ts.URL+"/dashboard")
	st := prop(t, page)["stats"].(map[string]any)
	if st["paid_count"].(float64) != 1 || st["total_spend"].(float64) != 75000 {
		t.Errorf("stats = %v", st)
	}
}

// ---- ulasan produk: guard sudah-beli-lunas, satu per item ----

func TestE2EProductReviews(t *testing.T) {
	_, ts, d := testServer(t)
	fileID := seedProduct(t, d, "produk-ulasan", 88000, "file")
	seedUser(t, d, "pembeli@uji.id", "rahasia123", "user")
	client := loginCookie(t, ts, "pembeli@uji.id", "rahasia123")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"pembeli@uji.id"}`, fileID))
	if res := paySimulasi(t, client, ts, orderNo, token); res.StatusCode != http.StatusFound {
		t.Fatalf("payNow = %d", res.StatusCode)
	}

	// guest (belum login) → form tak ditawarkan; POST ditolak guardVerified
	// (XHR Inertia = 409 + X-Inertia-Location /login)
	res := doPost(t, &http.Client{}, ts.URL+"/produk/produk-ulasan/ulasan", `{"rating":5}`)
	loc := res.Header.Get("X-Inertia-Location")
	if res.StatusCode == http.StatusFound {
		loc = res.Header.Get("Location")
	}
	if !strings.Contains(loc, "/login") {
		t.Errorf("ulasan guest = %d %s", res.StatusCode, loc)
	}

	// pembeli lunas mengulas 4 bintang
	csrf := csrfOf(t, client, ts, "/produk/produk-ulasan")
	res = doPostCSRF(t, client, ts.URL+"/produk/produk-ulasan/ulasan", `{"rating":4,"body":"Mantap"}`, csrf)
	if res.StatusCode != http.StatusFound || strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("ulasan = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// detail memuat ulasan + rata-rata
	page := doPage(t, client, ts.URL+"/produk/produk-ulasan")
	pp := prop(t, page)
	if pp["rating_count"].(float64) != 1 || pp["rating_avg"].(float64) != 4 {
		t.Errorf("rating = %v/%v, mau 1 ulasan rata 4", pp["rating_count"], pp["rating_avg"])
	}
	rev := pp["reviews"].([]any)[0].(map[string]any)
	if rev["body"] != "Mantap" || rev["rating"].(float64) != 4 {
		t.Errorf("review = %v", rev)
	}
	// sudah mengulas → can_review false (prop false dihilangkan gonertia)
	if got, ok := pp["can_review"]; ok && got == true {
		t.Errorf("can_review = %v, mau false setelah mengulas", got)
	}

	// ulas lagi → ditolak (entitlement habis)
	res = doPostCSRF(t, client, ts.URL+"/produk/produk-ulasan/ulasan", `{"rating":5}`, csrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("ulasan kedua = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	// rating di luar 1-5 → render ulang dengan errors (200)
	res = doPostCSRF(t, client, ts.URL+"/produk/produk-ulasan/ulasan", `{"rating":9}`, csrf)
	errs := postErrors(t, res)
	if errs["rating"] == "" {
		t.Errorf("rating 9 tanpa errors: %d %v", res.StatusCode, errs)
	}
}

// ---- tautan unduh di invoice/pay: satu tautan per item, multi item didukung ----

func TestE2EMultipleDownloadLinks(t *testing.T) {
	s, ts, d := testServer(t)
	// dua produk file dengan berkas nyata + satu file produk TANPA berkas
	f1 := seedProduct(t, d, "berkas-multi-1", 50000, "file")
	f2 := seedProduct(t, d, "berkas-multi-2", 75000, "file")
	fEmpty := seedProduct(t, d, "berkas-kosong", 10000, "file")
	for _, id := range []int64{f1, f2} {
		if err := writeFileForTest(t, s.filesDir(), fmt.Sprintf("berkas-%d.pdf", id), fmt.Sprintf("KONTEN-%d", id)); err != nil {
			t.Fatal(err)
		}
		d.Exec(`UPDATE products SET file_path=?, file_name=? WHERE id=?`, fmt.Sprintf("berkas-%d.pdf", id), fmt.Sprintf("Berkas %d.pdf", id), id)
	}

	// user checkout 3 produk file sekaligus (2 berkas + 1 kosong) dalam SATU order
	client := newClient()
	doPost(t, client, ts.URL+"/register", `{"name":"Multi","email":"multi@uji.id","password":"rahasia123"}`)
	verifyUser(t, s, d, client, ts, "multi@uji.id")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1},{"id":%d,"qty":1},{"id":%d,"qty":1}],"email":"multi@uji.id"}`, f1, f2, fEmpty))
	if res := paySimulasi(t, client, ts, orderNo, token); res.StatusCode != http.StatusFound {
		t.Fatalf("paySimulasi = %d", res.StatusCode)
	}

	// halaman pay (publik, ber-token): dua item punya has_download, item kosong tidak
	page := doPage(t, client, fmt.Sprintf("%s/pay/%s?t=%s", ts.URL, orderNo, token))
	items := prop(t, page)["order"].(map[string]any)["items"].([]any)
	downloads := 0
	for _, it := range items {
		if it.(map[string]any)["has_download"] == true {
			downloads++
		}
	}
	if downloads != 2 {
		t.Fatalf("item has_download = %d, mau 2 (produk tanpa berkas tak boleh punya tautan)", downloads)
	}

	// invoice: dua tautan /dl dengan item id berbeda (multi tautan)
	page = doPage(t, client, ts.URL+"/pesanan/"+orderNo)
	items = prop(t, page)["order"].(map[string]any)["items"].([]any)
	urls := map[string]bool{}
	for _, it := range items {
		m := it.(map[string]any)
		if m["has_download"] == true {
			urls[fmt.Sprintf("/dl/%s/%v?t=%s", orderNo, m["id"], token)] = true
		}
	}
	if len(urls) != 2 {
		t.Fatalf("tautan unduh unik di invoice = %d (%v), mau 2", len(urls), urls)
	}

	// kedua tautan benar-benar mengunduh isi masing-masing
	for u := range urls {
		res := doGetRes(t, &http.Client{}, ts.URL+u, nil)
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || !strings.Contains(string(b), "KONTEN-") {
			t.Errorf("unduh %s = %d %q", u, res.StatusCode, truncate(string(b), 40))
		}
	}
}

// ---- halaman checkout (GET) + gerbang verified ----

func TestE2ECheckoutPage(t *testing.T) {
	_, ts, d := testServer(t)
	seedUser(t, d, "shopper@uji.id", "rahasia123", "user")

	// guest → ditolak ke /login
	if res := doGetRes(t, newClient(), ts.URL+"/checkout", nil); res.StatusCode != http.StatusFound {
		t.Fatalf("checkout guest = %d, mau 302", res.StatusCode)
	}

	// user terdaftar tapi belum verifikasi email → gerbang 409 /verify-email
	unv := newClient()
	doPost(t, unv, ts.URL+"/register", `{"name":"Belum Verif","email":"unverif@uji.id","password":"rahasia123"}`)
	res := doGetRes(t, unv, ts.URL+"/checkout", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusConflict || res.Header.Get("X-Inertia-Location") != "/verify-email" {
		t.Fatalf("checkout belum verif = %d %s, mau 409 /verify-email", res.StatusCode, res.Header.Get("X-Inertia-Location"))
	}

	// user terverifikasi → halaman keranjang dirender
	page := doPage(t, loginCookie(t, ts, "shopper@uji.id", "rahasia123"), ts.URL+"/checkout")
	if page["component"] != "Public/Checkout" {
		t.Errorf("component = %v, mau Public/Checkout", page["component"])
	}
}
