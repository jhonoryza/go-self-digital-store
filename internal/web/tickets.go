package web

// Tiket bantuan (non-realtime): user buat tiket + balas, admin kelola.
// Status: open (menunggu admin) -> answered (admin sudah balas) -> closed.

import (
	"net/http"
	"strconv"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
)

type ticketData struct {
	ID        int64           `json:"id"`
	OrderNo   string          `json:"order_no"`
	Subject   string          `json:"subject"`
	Status    string          `json:"status"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	Messages  []ticketMessage `json:"messages,omitempty"`
}

type ticketMessage struct {
	ID        int64  `json:"id"`
	Sender    string `json:"sender"` // user | admin
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// listTickets tiket milik user (urut aktivitas terbaru).
func (s *Server) listTickets(userID int64) []ticketData {
	rows, err := s.DB.Query(`SELECT id, order_no, subject, status, created_at, updated_at
		FROM tickets WHERE user_id=? ORDER BY updated_at DESC LIMIT 100`, userID)
	if err != nil {
		logErr("listTickets: %v", err)
		return nil
	}
	defer rows.Close()
	var out []ticketData
	for rows.Next() {
		var t ticketData
		if rows.Scan(&t.ID, &t.OrderNo, &t.Subject, &t.Status, &t.CreatedAt, &t.UpdatedAt) == nil {
			out = append(out, t)
		}
	}
	return out
}

func (s *Server) ticketsPage(w http.ResponseWriter, r *http.Request, u auth.User) {
	s.Page(w, r, "Account/Tickets", inertia.Props{
		"tickets": s.listTickets(u.ID),
		"flash":   flashFromQuery(r),
	})
}

// loadTicket detail tiket + pesan; ok=false bila bukan milik user.
func (s *Server) loadTicket(id int64, u auth.User) (ticketData, bool) {
	var t ticketData
	var owner int64
	err := s.DB.QueryRow(`SELECT id, user_id, order_no, subject, status, created_at, updated_at
		FROM tickets WHERE id=?`, id).Scan(&t.ID, &owner, &t.OrderNo, &t.Subject, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil || (owner != u.ID && u.Role != "admin") {
		return t, false
	}
	msgs, _ := s.DB.Query(`SELECT id, sender, body, created_at FROM ticket_messages WHERE ticket_id=? ORDER BY id`, id)
	if msgs != nil {
		defer msgs.Close()
		for msgs.Next() {
			var m ticketMessage
			if msgs.Scan(&m.ID, &m.Sender, &m.Body, &m.CreatedAt) == nil {
				t.Messages = append(t.Messages, m)
			}
		}
	}
	return t, true
}

func (s *Server) ticketDetailPage(w http.ResponseWriter, r *http.Request, u auth.User) {
	id, err := pathInt(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	t, ok := s.loadTicket(id, u)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.Page(w, r, "Account/TicketDetail", inertia.Props{
		"ticket": t,
		"flash":  flashFromQuery(r),
	})
}

// createTicket POST /tiket — tiket baru (boleh terkait order milik sendiri).
func (s *Server) createTicket(w http.ResponseWriter, r *http.Request, u auth.User) {
	if ipLimited("ticket", s.clientIP(r)) {
		jsonError(w, http.StatusTooManyRequests, "terlalu banyak permintaan")
		return
	}
	subject := strings.TrimSpace(r.PostFormValue("subject"))
	body := strings.TrimSpace(r.PostFormValue("body"))
	orderNo := strings.TrimSpace(r.PostFormValue("order_no"))

	errs := inertia.ValidationErrors{}
	if subject == "" {
		errs["subject"] = "subjek wajib diisi"
	}
	if body == "" {
		errs["body"] = "pesan wajib diisi"
	}
	if orderNo != "" && !s.canAccessOrder(orderNo, u) {
		errs["order_no"] = "order tidak ditemukan di akun Anda"
	}
	if len(errs) > 0 {
		s.renderWithErrors(w, r, "Account/Tickets", nil, errs)
		return
	}

	res, err := s.DB.Exec(`INSERT INTO tickets (user_id, order_no, subject) VALUES (?,?,?)`, u.ID, orderNo, subject)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	tid, _ := res.LastInsertId()
	s.DB.Exec(`INSERT INTO ticket_messages (ticket_id, sender, body) VALUES (?,?,?)`, tid, "user", body)
	s.flashOK(w, r, "/tiket/"+fmtInt(tid), "Tiket dibuat - admin akan membalas di halaman ini.")
}

// replyTicket POST /tiket/{id} — balasan user; status kembali 'open'.
func (s *Server) replyTicket(w http.ResponseWriter, r *http.Request, u auth.User) {
	id, err := pathInt(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	t, ok := s.loadTicket(id, u)
	if !ok || t.Status == "closed" {
		if t.Status == "closed" {
			s.flashErr(w, r, "/tiket/"+fmtInt(id), "Tiket sudah ditutup - buka tiket baru bila perlu.")
			return
		}
		http.NotFound(w, r)
		return
	}
	body := strings.TrimSpace(r.PostFormValue("body"))
	if body == "" {
		s.renderWithErrors(w, r, "Account/TicketDetail", nil, inertia.ValidationErrors{"body": "pesan wajib diisi"})
		return
	}
	s.DB.Exec(`INSERT INTO ticket_messages (ticket_id, sender, body) VALUES (?,?,?)`, id, "user", body)
	s.DB.Exec(`UPDATE tickets SET status='open', updated_at=datetime('now') WHERE id=?`, id)
	s.Redirect(w, r, "/tiket/"+fmtInt(id))
}

// pathInt konversi path param int.
func pathInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func fmtInt(n int64) string {
	return strconv.FormatInt(n, 10)
}
