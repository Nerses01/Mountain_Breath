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

// addressPayload is the one JSON shape an address has, in both directions:
// the checkout request carries it up, the order response carries the frozen
// snapshot back down. Sharing the struct is safe because the fields really
// are the same — what differs is mutability, and JSON has no opinion on that.
type addressPayload struct {
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Phone      string `json:"phone"`
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

func toAddressPayload(a domain.Address) addressPayload {
	return addressPayload{
		FirstName: a.FirstName, LastName: a.LastName, Phone: a.Phone,
		Street: a.Street, City: a.City, PostalCode: a.PostalCode, Country: a.Country,
	}
}

func (p addressPayload) toDomain() domain.Address {
	return domain.Address{
		FirstName: p.FirstName, LastName: p.LastName, Phone: p.Phone,
		Street: p.Street, City: p.City, PostalCode: p.PostalCode, Country: p.Country,
	}
}

type orderResponse struct {
	ID        int64               `json:"id"`
	Status    string              `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
	UserEmail string              `json:"user_email,omitempty"` // admin view only
	Items     []orderItemResponse `json:"items"`

	// An order carries ONE currency and no second price, which is the
	// deliberate difference from every other money-bearing response in this
	// API. A cart can be read in either market; an order is a record of what
	// was actually charged, and showing a converted alternative next to it
	// would invite "but you charged me the other number".
	Currency domain.Currency `json:"currency"`
	// A decimal as a STRING (see domain.Order.FxRateUsed): JSON numbers are
	// doubles in every mainstream parser, and this one is NUMERIC(18,8).
	// Omitted entirely for a base-currency order, where no rate applied.
	FxRateUsed *string `json:"fx_rate_used,omitempty"`

	// E6: the five-figure breakdown the design's summary card draws.
	// total_minor keeps its Phase 5 name and meaning — it IS the grand
	// total — so every client written before E6 keeps working.
	SubtotalMinor int64 `json:"subtotal_minor"`
	ShippingMinor int64 `json:"shipping_minor"`
	DiscountMinor int64 `json:"discount_minor"`
	// E7: the composition of discount_minor (member + promo = discount),
	// because the receipt draws them as separate lines; promo_code is the
	// frozen text of the code that was redeemed, absent when none was.
	MemberDiscountMinor int64  `json:"member_discount_minor"`
	PromoDiscountMinor  int64  `json:"promo_discount_minor"`
	PromoCode           string `json:"promo_code,omitempty"`
	// Contained in the subtotal ("Prices include VAT"), never added on top;
	// shown on the invoice line, absent from the arithmetic.
	TaxMinor   int64 `json:"tax_minor"`
	TotalMinor int64 `json:"total_minor"`

	PaymentMethod string `json:"payment_method"`
	PaymentStatus string `json:"payment_status"`

	// The frozen snapshot. A pointer so pre-E6 orders honestly send null
	// rather than seven empty strings pretending to be an address.
	ShipTo             *addressPayload `json:"ship_to,omitempty"`
	DeliveryNote       string          `json:"delivery_note,omitempty"`
	LeaveWithNeighbour bool            `json:"leave_with_neighbour"`
}

func toOrderResponse(o domain.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemResponse{
			Name: it.Name, Label: it.Label, PriceMinor: it.PriceMinor, Qty: it.Qty,
		})
	}
	resp := orderResponse{
		ID: o.ID, Status: o.Status,
		CreatedAt: o.CreatedAt, UserEmail: o.UserEmail, Items: items,
		Currency: o.Currency, FxRateUsed: o.FxRateUsed,
		SubtotalMinor: o.Totals.SubtotalMinor, ShippingMinor: o.Totals.ShippingMinor,
		DiscountMinor: o.Totals.DiscountMinor, TaxMinor: o.Totals.TaxMinor,
		MemberDiscountMinor: o.Totals.MemberDiscountMinor,
		PromoDiscountMinor:  o.Totals.PromoDiscountMinor,
		PromoCode:           o.PromoCode,
		TotalMinor:          o.TotalMinor,
		PaymentMethod: o.PaymentMethod, PaymentStatus: o.PaymentStatus,
		DeliveryNote: o.DeliveryNote, LeaveWithNeighbour: o.LeaveWithNeighbour,
	}
	if o.ShipTo != nil {
		ship := toAddressPayload(*o.ShipTo)
		resp.ShipTo = &ship
	}
	return resp
}

// checkoutRequest is everything the client CONTRIBUTES to an order — and
// there is no money in it, which is the security design of the endpoint.
// DisallowUnknownFields turns a body that tries to send "total_minor" into a
// 400 before any handler code runs: the server does not ignore a
// client-supplied total, it refuses the request outright.
//
// Note what else is absent: card numbers. The design draws card fields, but
// this API never accepts them — card payment is a stub until a real
// provider (Phase 11) takes the card data on ITS servers. An API that
// accepts card numbers "just to a stub" has put itself in scope for PCI
// compliance for nothing.
type checkoutRequest struct {
	Address            addressPayload `json:"address"`
	PaymentMethod      string         `json:"payment_method"`
	DeliveryNote       string         `json:"delivery_note"`
	LeaveWithNeighbour bool           `json:"leave_with_neighbour"`
}

// POST /orders — checkout: the current cart becomes an order.
func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req checkoutRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	// The currency comes from the EDGE, never from the request body. A client
	// that could name what it pays in could name the cheaper of two markets
	// for a basket priced in the dearer one.
	currency := currencyFromContext(r.Context())

	in := domain.CheckoutInput{
		Address:            req.Address.toDomain(),
		PaymentMethod:      req.PaymentMethod,
		DeliveryNote:       req.DeliveryNote,
		LeaveWithNeighbour: req.LeaveWithNeighbour,
	}
	if fields := domain.ValidateCheckout(in, currency); len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	order, err := s.store.CreateOrder(r.Context(), user.ID, currency, in)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmptyCart):
			s.respondError(w, http.StatusConflict, "empty_cart", "your cart is empty")
		case errors.Is(err, domain.ErrInsufficientStock):
			// err carries the product name; safe and useful for the customer
			s.respondError(w, http.StatusConflict, "insufficient_stock", err.Error())
		case errors.Is(err, domain.ErrPriceUnavailable):
			// 409, not 500: the shop is misconfigured for this market, not
			// broken. The message names the product so the customer can drop
			// it or switch currency, and so the family can fix the price.
			s.respondError(w, http.StatusConflict, "price_unavailable", err.Error())
		case errors.Is(err, domain.ErrPromoInvalid):
			// The cart's code stopped being valid between apply and "Place
			// the order". Refused, not silently repriced — the customer saw
			// a total this order would no longer match. The client refreshes
			// its preview, which names the reason inline.
			s.respondError(w, http.StatusConflict, "promo_invalid", err.Error())
		default:
			s.log.Error("creating order", "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	s.metrics.ordersCreated.Inc()
	s.respondJSON(w, http.StatusCreated, toOrderResponse(order))
}

// GET /orders/{id} — the confirmation page's read. Owner or admin.
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	orderID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "order id must be a number")
		return
	}

	order, err := s.store.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such order")
			return
		}
		s.log.Error("getting order", "id", orderID, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// 404 for someone else's order, deliberately NOT 403. The E4 rule ran
	// the other way (a non-purchaser reviewing gets 403) because there the
	// resource was public and only the ACTION was denied. An order is
	// private data: a 403 would confirm to whoever is enumerating ids that
	// order 12 exists and belongs to somebody — which is exactly the
	// information a URL guesser is fishing for.
	if order.UserID != user.ID && user.Role != domain.RoleAdmin {
		s.respondError(w, http.StatusNotFound, "not_found", "no such order")
		return
	}

	s.respondJSON(w, http.StatusOK, toOrderResponse(order))
}

// GET /account/address — the saved address for pre-filling the checkout
// form. A first-time customer gets 404, which the client renders as an
// empty form, not an error.
func (s *Server) handleGetDefaultAddress(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	addr, err := s.store.DefaultAddress(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no saved address")
			return
		}
		s.log.Error("getting default address", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toAddressPayload(addr))
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
