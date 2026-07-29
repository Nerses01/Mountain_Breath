package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

type orderItemResponse struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	PriceMinor int64  `json:"price_minor"`
	Qty        int    `json:"qty"`
}

type orderResponse struct {
	ID         int64               `json:"id"`
	Status     string              `json:"status"`
	TotalMinor int64               `json:"total_minor"`
	CreatedAt  time.Time           `json:"created_at"`
	UserEmail  string              `json:"user_email,omitempty"` // admin view only
	Items      []orderItemResponse `json:"items"`
}

func toOrderResponse(o domain.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemResponse{
			Name: it.Name, Label: it.Label, PriceMinor: it.PriceMinor, Qty: it.Qty,
		})
	}
	return orderResponse{
		ID: o.ID, Status: o.Status, TotalMinor: o.TotalMinor,
		CreatedAt: o.CreatedAt, UserEmail: o.UserEmail, Items: items,
	}
}

// POST /orders — checkout: the current cart becomes an order.
func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	order, err := s.store.CreateOrder(r.Context(), user.ID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmptyCart):
			s.respondError(w, http.StatusConflict, "empty_cart", "your cart is empty")
		case errors.Is(err, domain.ErrInsufficientStock):
			// err carries the product name; safe and useful for the customer
			s.respondError(w, http.StatusConflict, "insufficient_stock", err.Error())
		default:
			s.log.Error("creating order", "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	s.respondJSON(w, http.StatusCreated, toOrderResponse(order))
}

func (s *Server) handleListMyOrders(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	orders, err := s.store.ListOrdersByUser(r.Context(), user.ID)
	if err != nil {
		s.log.Error("listing orders", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, toOrderResponse(o))
	}
	s.respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.store.ListAllOrders(r.Context())
	if err != nil {
		s.log.Error("listing all orders", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, toOrderResponse(o))
	}
	s.respondJSON(w, http.StatusOK, resp)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// PATCH /admin/orders/{id}/status — drive the order state machine.
func (s *Server) handleUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "order id must be a number")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req updateStatusRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	if !domain.ValidOrderStatus(req.Status) {
		s.respondValidationError(w, map[string]string{"status": "unknown status"})
		return
	}

	order, err := s.store.UpdateOrderStatus(r.Context(), orderID, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			s.respondError(w, http.StatusNotFound, "not_found", "no such order")
		case errors.Is(err, domain.ErrInvalidTransition):
			s.respondError(w, http.StatusConflict, "invalid_transition", err.Error())
		default:
			s.log.Error("updating order status", "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	s.respondJSON(w, http.StatusOK, toOrderResponse(order))
}
