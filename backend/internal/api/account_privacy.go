package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// F2 (decision #97): the privacy page's two promises made executable —
// "we will show you what we store about you" (GET /account/data) and
// "delete your account entirely" (DELETE /account). The page's own
// carve-out is the deletion's contract: "Orders we must keep for
// bookkeeping as the law requires; everything else goes."

// accountDataResponse is the export: every table that knows this person,
// in one JSON a human can read. Composed from the SAME store reads the
// screens use — an export assembled from special queries could drift from
// what the shop actually shows and acts on.
type accountDataResponse struct {
	Account struct {
		Email              string    `json:"email"`
		FullName           string    `json:"full_name,omitempty"`
		Phone              string    `json:"phone,omitempty"`
		Role               string    `json:"role"`
		CreatedAt          time.Time `json:"created_at"`
		NotifyOrderUpdates bool      `json:"notify_order_updates"`
	} `json:"account"`
	Addresses []addressEntryPayload `json:"addresses"`
	Orders    []orderResponse       `json:"orders"`
	Wishlist  []wishlistDataItem    `json:"wishlist"`
	Reviews   []reviewDataItem      `json:"reviews"`
	// The three-state newsletter fact ("none" covers never and
	// unsubscribed — the same restartable state).
	Newsletter string `json:"newsletter"`
	// Devices currently holding a live key to the account.
	ActiveSessions int `json:"active_sessions"`
}

type wishlistDataItem struct {
	Slug    string    `json:"slug"`
	Name    string    `json:"name"`
	SavedAt time.Time `json:"saved_at"`
}

type reviewDataItem struct {
	ProductSlug string    `json:"product_slug"`
	Rating      int       `json:"rating"`
	Title       string    `json:"title,omitempty"`
	Body        string    `json:"body,omitempty"`
	// Every status, pending and rejected included — they are still this
	// person's words, and an export that hid them would lie by omission.
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// GET /account/data — the data view.
func (s *Server) handleAccountData(w http.ResponseWriter, r *http.Request) {
	ctxUser, _ := userFrom(r.Context())

	user, err := s.store.GetUserByID(r.Context(), ctxUser.ID)
	if err != nil {
		s.log.Error("reading account for export", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	var resp accountDataResponse
	resp.Account.Email = user.Email
	resp.Account.FullName = user.FullName
	resp.Account.Phone = user.Phone
	resp.Account.Role = user.Role
	resp.Account.CreatedAt = user.CreatedAt
	resp.Account.NotifyOrderUpdates = user.NotifyOrderUpdates

	addresses, err := s.store.ListAddresses(r.Context(), user.ID)
	if err != nil {
		s.log.Error("listing addresses for export", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp.Addresses = make([]addressEntryPayload, 0, len(addresses))
	for _, a := range addresses {
		resp.Addresses = append(resp.Addresses, toAddressEntryPayload(a))
	}

	orders, err := s.store.ListOrdersByUser(r.Context(), user.ID)
	if err != nil {
		s.log.Error("listing orders for export", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp.Orders = make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp.Orders = append(resp.Orders, toOrderResponse(o))
	}

	wishlist, err := s.store.ListWishlist(r.Context(), user.ID, viewFromContext(r.Context()))
	if err != nil {
		s.log.Error("listing wishlist for export", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp.Wishlist = make([]wishlistDataItem, 0, len(wishlist))
	for _, item := range wishlist {
		resp.Wishlist = append(resp.Wishlist, wishlistDataItem{
			Slug: item.Slug, Name: item.Name, SavedAt: item.SavedAt,
		})
	}

	reviews, slugs, err := s.store.ReviewsByUser(r.Context(), user.ID)
	if err != nil {
		s.log.Error("listing reviews for export", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp.Reviews = make([]reviewDataItem, 0, len(reviews))
	for i, rev := range reviews {
		resp.Reviews = append(resp.Reviews, reviewDataItem{
			ProductSlug: slugs[i], Rating: rev.Rating, Title: rev.Title,
			Body: rev.Body, Status: rev.Status, CreatedAt: rev.CreatedAt,
		})
	}

	resp.Newsletter, err = s.store.NewsletterStatusByEmail(r.Context(), user.Email)
	if err != nil {
		s.log.Error("reading newsletter status for export", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp.ActiveSessions, err = s.store.CountSessions(r.Context(), user.ID)
	if err != nil {
		s.log.Error("counting sessions for export", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, resp)
}

type deleteAccountRequest struct {
	CurrentPassword string `json:"current_password"`
}

// DELETE /account — the point of no return, so it re-authenticates: the
// current password for password accounts (a stolen session must not be
// enough to erase someone), and nothing extra for OAuth-only accounts —
// their hash is empty by construction, there IS no password to ask for,
// and their session is exactly the credential Google vouched for.
func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req deleteAccountRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	if user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword(
			[]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			s.respondValidationError(w, map[string]string{
				"current_password": domain.ValidationIncorrectPassword,
			})
			return
		}
	}

	if err := s.store.DeleteAccount(r.Context(), user.ID); err != nil {
		switch {
		case errors.Is(err, domain.ErrLastAdmin):
			s.respondError(w, http.StatusConflict, "last_admin",
				"you are the only admin — promote someone else before deleting this account")
		case errors.Is(err, domain.ErrNotFound):
			s.respondError(w, http.StatusNotFound, "not_found", "no such account")
		default:
			s.log.Error("deleting account", "id", user.ID, "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	// The sessions died with the account (CASCADE); this clears the
	// browser's now-dangling cookie the same way logout does.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !s.devMode,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}
