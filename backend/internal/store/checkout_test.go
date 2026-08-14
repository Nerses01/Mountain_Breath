package store_test

import (
	"context"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// The address snapshot must survive the customer editing their address —
// the same promise price snapshots made in Phase 5, extended to where the
// parcel went.
func TestCreateOrder_AddressSnapshotSurvivesAnEdit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 1400})
	userID := seedUserWithCart(t, "mover@test.local", variantID, 1)

	in := testCheckout()
	in.Address.Street = "14 Abovyan St, apt 6"
	order, err := s.CreateOrder(ctx, userID, domain.CurrencyUSD, in)
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// The checkout saved the address book row; the customer then MOVES.
	if _, err := testPool.Exec(ctx,
		`UPDATE addresses SET street = '2 Mashtots Ave' WHERE user_id = $1`, userID); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if got.ShipTo == nil || got.ShipTo.Street != "14 Abovyan St, apt 6" {
		t.Errorf("order address followed the edit: %+v", got.ShipTo)
	}

	// ...while the BOOK shows the new street, ready to pre-fill the next
	// checkout. Two answers, both right, because they are different questions.
	addr, err := s.DefaultAddress(ctx, userID)
	if err != nil {
		t.Fatalf("DefaultAddress: %v", err)
	}
	if addr.Street != "2 Mashtots Ave" {
		t.Errorf("address book street = %q, want the edited one", addr.Street)
	}
}

// A second checkout REPLACES the default address rather than accumulating
// rows — one default per user is a partial unique index, and the upsert
// names it as its arbiter.
func TestCheckout_UpsertsOneDefaultAddress(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 1400})
	userID := seedUserWithCart(t, "repeat@test.local", variantID, 1)

	first := testCheckout()
	first.Address.City = "Yerevan"
	if _, err := s.CreateOrder(ctx, userID, domain.CurrencyUSD, first); err != nil {
		t.Fatal(err)
	}

	// Second order, new city.
	if _, err := testPool.Exec(ctx,
		`INSERT INTO cart_items (user_id, variant_id, qty) VALUES ($1, $2, 1)`,
		userID, variantID); err != nil {
		t.Fatal(err)
	}
	second := testCheckout()
	second.Address.City = "Gyumri"
	if _, err := s.CreateOrder(ctx, userID, domain.CurrencyUSD, second); err != nil {
		t.Fatal(err)
	}

	var rows int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM addresses WHERE user_id = $1`, userID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Errorf("%d address rows, want 1", rows)
	}
	addr, err := s.DefaultAddress(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if addr.City != "Gyumri" {
		t.Errorf("default city = %q, want the second checkout's", addr.City)
	}
}

func TestCreateOrder_CashOnDeliveryLandsUnpaid(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{
		domain.CurrencyUSD: 1400, domain.CurrencyAMD: 6700,
	})
	userID := seedUserWithCart(t, "cod@test.local", variantID, 1)

	in := testCheckout()
	in.PaymentMethod = domain.PayCashOnDelivery
	order, err := s.CreateOrder(ctx, userID, domain.CurrencyAMD, in)
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	if order.PaymentMethod != domain.PayCashOnDelivery {
		t.Errorf("method = %q", order.PaymentMethod)
	}
	// The plan's test, by name: choosing cash records an intention, not a
	// payment. Money changes hands at the door, and the admin records THAT.
	if order.PaymentStatus != domain.PaymentUnpaid {
		t.Errorf("status = %q, want unpaid", order.PaymentStatus)
	}
}

// One chilled jar anywhere in the basket adds the surcharge to the parcel —
// and the surcharge survives free shipping, which the base does not.
func TestCreateOrder_ColdChainSurcharge(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	// A chilled product (royal jelly's flag from E3) dear enough that one
	// jar clears the $70 free-shipping threshold on its own.
	jellyID := seedPricedProduct(t, "jelly", "RJL-1", 10, domain.Money{domain.CurrencyUSD: 9000})
	if _, err := testPool.Exec(ctx,
		`UPDATE products SET is_cold_chain = TRUE WHERE slug = 'jelly'`); err != nil {
		t.Fatal(err)
	}
	userID := seedUserWithCart(t, "chilled@test.local", jellyID, 1)

	order, err := s.CreateOrder(ctx, userID, domain.CurrencyUSD, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// $90 subtotal: base ($4) waived by the threshold, surcharge ($6) not.
	if order.Totals.ShippingMinor != 600 {
		t.Errorf("shipping = %d, want 600 (surcharge only)", order.Totals.ShippingMinor)
	}
	if order.TotalMinor != 9600 {
		t.Errorf("total = %d, want 9600", order.TotalMinor)
	}
	// The contained VAT rides along informationally: 9000 × 20/120 = 1500.
	if order.Totals.TaxMinor != 1500 {
		t.Errorf("tax = %d, want 1500", order.Totals.TaxMinor)
	}
}

// The admin listing joins users onto the shared column list — the one order
// read whose SQL SHAPE differs from the others, which is exactly where a
// shared constant can hide a query that only breaks in the variant. It did:
// unqualified columns were ambiguous against users.id, every /admin/orders
// request 500ed in the running shop, and nothing here noticed because no
// integration test called ListAllOrders. Now one does.
func TestListAllOrders_JoinsTheCustomerEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 1400})
	userID := seedUserWithCart(t, "admin-view@test.local", variantID, 1)
	placed, err := s.CreateOrder(ctx, userID, domain.CurrencyUSD, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	orders, err := s.ListAllOrders(ctx)
	if err != nil {
		t.Fatalf("ListAllOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("got %d orders, want 1", len(orders))
	}

	got := orders[0]
	if got.ID != placed.ID {
		t.Errorf("order id = %d, want %d", got.ID, placed.ID)
	}
	// The join's whole purpose — and the field only this read populates.
	if got.UserEmail != "admin-view@test.local" {
		t.Errorf("user email = %q, want the customer's", got.UserEmail)
	}
	// The E6 columns survive the JOIN-shaped variant of the query too.
	if got.ShipTo == nil || got.ShipTo.City != "Yerevan" {
		t.Errorf("address snapshot missing from the admin view: %+v", got.ShipTo)
	}
	if got.PaymentStatus != domain.PaymentUnpaid {
		t.Errorf("payment status = %q", got.PaymentStatus)
	}
	if len(got.Items) != 1 {
		t.Errorf("items not attached: %+v", got.Items)
	}
}

// GetCart's quote and CreateOrder's charge use the same arithmetic — the
// number the customer read IS the number they are charged.
func TestCartQuoteMatchesTheCharge(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "pollen", "POL-1", 10, domain.Money{
		domain.CurrencyUSD: 1600, domain.CurrencyAMD: 7600,
	})
	userID := seedUserWithCart(t, "quote@test.local", variantID, 2)

	items, err := s.GetCart(ctx, userID, domain.View{Currency: domain.CurrencyAMD})
	if err != nil {
		t.Fatal(err)
	}
	rates, err := s.ShippingRates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	quote := domain.QuoteCart(items, rates)

	order, err := s.CreateOrder(ctx, userID, domain.CurrencyAMD, testCheckout())
	if err != nil {
		t.Fatal(err)
	}

	if quote.TotalMinor[domain.CurrencyAMD] != order.TotalMinor {
		t.Errorf("quoted %d, charged %d — the customer read a number the shop did not honour",
			quote.TotalMinor[domain.CurrencyAMD], order.TotalMinor)
	}
}
