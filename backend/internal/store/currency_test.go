package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// TestCurrenciesMatchTheDatabase is the tripwire under domain.Currencies.
//
// The Go set and the `currencies` table are the same fact written twice —
// the same duplication domain.Locales has, and E1.5 could only paper over it
// with a comment asking future readers to keep three places in sync. A
// comment is a wish. This is the version that fails a build.
func TestCurrenciesMatchTheDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	ctx := context.Background()

	rows, err := testPool.Query(ctx,
		`SELECT code FROM currencies WHERE is_active ORDER BY sort_order`)
	if err != nil {
		t.Fatalf("querying currencies: %v", err)
	}
	defer rows.Close()

	var inDB []domain.Currency
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			t.Fatal(err)
		}
		inDB = append(inDB, domain.Currency(code))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(inDB) != len(domain.Currencies) {
		t.Fatalf("database has %v, domain.Currencies has %v", inDB, domain.Currencies)
	}
	for i := range inDB {
		if inDB[i] != domain.Currencies[i] {
			t.Errorf("position %d: database says %q, domain.Currencies says %q "+
				"(order matters — it is the order the footer switcher lists them in)",
				i, inDB[i], domain.Currencies[i])
		}
	}

	// E8 widened the duplication: emails are rendered in Go, so the minor
	// exponent gained a Go copy too — same tripwire, one more column.
	for _, c := range domain.Currencies {
		var dbExp int
		if err := testPool.QueryRow(ctx,
			`SELECT minor_exponent FROM currencies WHERE code = $1`, string(c)).Scan(&dbExp); err != nil {
			t.Fatalf("querying exponent for %s: %v", c, err)
		}
		if dbExp != c.MinorExponent() {
			t.Errorf("%s: database exponent %d, Go says %d — FormatMinor would "+
				"misplace the decimal point in every email", c, dbExp, c.MinorExponent())
		}
	}

	// The base currency is the one price the schema treats as mandatory, so
	// disagreeing about which it is would be worse than disagreeing about
	// the set.
	var dbBase string
	if err := testPool.QueryRow(ctx, `SELECT code FROM currencies WHERE is_base`).Scan(&dbBase); err != nil {
		t.Fatalf("querying base currency: %v", err)
	}
	if domain.Currency(dbBase) != domain.BaseCurrency {
		t.Errorf("database base = %q, domain.BaseCurrency = %q", dbBase, domain.BaseCurrency)
	}
}

// seedPricedProduct inserts one product with one variant priced in the given
// markets, and returns the variant id.
func seedPricedProduct(t *testing.T, slug, sku string, stock int, prices domain.Money) int64 {
	t.Helper()
	ctx := context.Background()

	var variantID int64
	err := testPool.QueryRow(ctx, `
		WITH cat AS (
			INSERT INTO categories (slug, name) VALUES ($1 || '-cat', 'Cat')
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		), prod AS (
			INSERT INTO products (category_id, slug, name)
			SELECT id, $1, $1 FROM cat RETURNING id
		)
		INSERT INTO product_variants (product_id, sku, label, stock_qty)
		SELECT id, $2, '500 g', $3 FROM prod
		RETURNING id`, slug, sku, stock).Scan(&variantID)
	if err != nil {
		t.Fatalf("seeding product %q: %v", slug, err)
	}

	for currency, minor := range prices {
		if _, err := testPool.Exec(ctx, `
			INSERT INTO variant_prices (variant_id, currency, price_minor)
			VALUES ($1, $2, $3)`, variantID, currency, minor); err != nil {
			t.Fatalf("pricing %q in %s: %v", slug, currency, err)
		}
	}
	return variantID
}

func priceIn(t *testing.T, variantID int64, currency domain.Currency) (int64, bool) {
	t.Helper()
	var minor int64
	err := testPool.QueryRow(context.Background(), `
		SELECT price_minor FROM variant_effective_prices
		WHERE variant_id = $1 AND currency = $2`, variantID, currency).Scan(&minor)
	if err != nil {
		return 0, false
	}
	return minor, true
}

// An explicit shelf price is a business decision and must survive contact
// with the exchange rate.
func TestEffectivePrice_ShelfPriceBeatsConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)

	id := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{
		domain.CurrencyUSD: 1400,
		domain.CurrencyAMD: 6700, // NOT 1400 × 3.90 = 5460
	})

	got, ok := priceIn(t, id, domain.CurrencyAMD)
	if !ok {
		t.Fatal("no AMD price at all")
	}
	if got != 6700 {
		t.Errorf("AMD price = %d, want the shelf price 6700 (5460 would mean the rate won)", got)
	}
}

func TestEffectivePrice_ConvertsAndSnapsToTheRoundingStep(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)

	// $14.99 with no dram price on file. 1499 × 3.90 = 5,846.1 drams, which
	// AMD's rounding_step of 10 turns into 5,850 — a number a shop could
	// write on a label, which is the entire reason the column exists.
	id := seedPricedProduct(t, "propolis", "PRO-1", 10, domain.Money{domain.CurrencyUSD: 1499})

	got, ok := priceIn(t, id, domain.CurrencyAMD)
	if !ok {
		t.Fatal("conversion produced no AMD price")
	}
	if got != 5850 {
		t.Errorf("converted AMD price = %d, want 5850", got)
	}
	if got%10 != 0 {
		t.Errorf("converted price %d is not a multiple of the rounding step", got)
	}

	// The minor-unit SCALE changes too, not just the number: 1499 is $14.99
	// (two decimals) while 5850 is 5,850 drams (none). Anything that divided
	// by 100 here would be out by two orders of magnitude.
	var exponent int
	if err := testPool.QueryRow(context.Background(),
		`SELECT minor_exponent FROM currencies WHERE code = 'AMD'`).Scan(&exponent); err != nil {
		t.Fatal(err)
	}
	if exponent != 0 {
		t.Errorf("AMD minor_exponent = %d, want 0", exponent)
	}
}

// The catalog's price sort, price filter and slider bounds are all answers
// to "in WHICH market?" — and per-market pricing means the answers genuinely
// differ. This is the test that would have caught treating currency as a
// display-only concern.
func TestCatalog_PriceOrderDiffersBetweenMarkets(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	// Cheaper in dollars, dearer in drams — and the other way round.
	seedPricedProduct(t, "alpha", "A-1", 10, domain.Money{domain.CurrencyUSD: 1000, domain.CurrencyAMD: 5000})
	seedPricedProduct(t, "beta", "B-1", 10, domain.Money{domain.CurrencyUSD: 1200, domain.CurrencyAMD: 4000})

	list := func(c domain.Currency, min, max *int64) []string {
		t.Helper()
		products, _, err := s.ListProducts(ctx, domain.ProductFilter{
			Page: 1, PerPage: 10, Sort: domain.SortPriceAsc, Currency: c,
			PriceMinMinor: min, PriceMaxMinor: max,
		})
		if err != nil {
			t.Fatalf("ListProducts(%s): %v", c, err)
		}
		return slugsOf(products)
	}

	if got := list(domain.CurrencyUSD, nil, nil); !equalSlugs(got, "alpha", "beta") {
		t.Errorf("cheapest-first in USD = %v, want [alpha beta]", got)
	}
	if got := list(domain.CurrencyAMD, nil, nil); !equalSlugs(got, "beta", "alpha") {
		t.Errorf("cheapest-first in AMD = %v, want [beta alpha]", got)
	}

	// The filter bounds are in the resolved currency's minor units, so the
	// same number means different money in each market.
	ceiling := int64(1100)
	if got := list(domain.CurrencyUSD, nil, &ceiling); !equalSlugs(got, "alpha") {
		t.Errorf("USD ≤ 1100 = %v, want [alpha]", got)
	}
	ceiling = 4500
	if got := list(domain.CurrencyAMD, nil, &ceiling); !equalSlugs(got, "beta") {
		t.Errorf("AMD ≤ 4500 = %v, want [beta]", got)
	}

	// ...and so are the slider's ends.
	facets, err := s.CatalogFacets(ctx, domain.ProductFilter{Currency: domain.CurrencyAMD})
	if err != nil {
		t.Fatalf("CatalogFacets: %v", err)
	}
	if facets.PriceMinMinor != 4000 || facets.PriceMaxMinor != 5000 {
		t.Errorf("AMD facet bounds = %d–%d, want 4000–5000",
			facets.PriceMinMinor, facets.PriceMaxMinor)
	}
}

func TestGetCart_CarriesEveryMarket(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "jelly", "RJL-1", 10, domain.Money{
		domain.CurrencyUSD: 3200, domain.CurrencyAMD: 15300,
	})
	userID := seedUserWithCart(t, "dual@test.local", variantID, 2)

	items, err := s.GetCart(ctx, userID, domain.View{Currency: domain.CurrencyAMD})
	if err != nil {
		t.Fatalf("GetCart: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d cart lines, want 1", len(items))
	}

	// The resolved currency drives the arithmetic...
	if items[0].PriceMinor != 15300 || items[0].LineTotalMinor() != 30600 {
		t.Errorf("AMD line = %d × %d, want 15300 × 2", items[0].PriceMinor, items[0].Qty)
	}
	// ...and the other market rides along for the design's second line.
	if items[0].Prices[domain.CurrencyUSD] != 3200 {
		t.Errorf("USD price missing from the line: %v", items[0].Prices)
	}
}

func TestCreateOrder_SnapshotsTheCurrencyAndTheRate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "pollen", "POL-1", 10, domain.Money{
		domain.CurrencyUSD: 1600, domain.CurrencyAMD: 7600,
	})
	userID := seedUserWithCart(t, "amd-buyer@test.local", variantID, 3)

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyAMD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	if order.Currency != domain.CurrencyAMD {
		t.Errorf("order currency = %q, want AMD", order.Currency)
	}
	// Charged from the AMD shelf price — 7,600 × 3, not the dollar price
	// converted. No shipping: this is the buyer's FIRST order, and E7's
	// hive-club perk waives the base (the per-market rate card itself is
	// pinned by TestHiveClubPerksAcrossTwoOrders, on an order that pays it).
	if order.Totals.SubtotalMinor != 22800 {
		t.Errorf("subtotal = %d, want 22800", order.Totals.SubtotalMinor)
	}
	if order.TotalMinor != 22800 {
		t.Errorf("total = %d, want 22800 (first delivery free)", order.TotalMinor)
	}
	if order.Items[0].PriceMinor != 7600 {
		t.Errorf("item snapshot = %d, want 7600", order.Items[0].PriceMinor)
	}
	if order.FxRateUsed == nil {
		t.Fatal("fx_rate_used is nil for a non-base order")
	}
	if *order.FxRateUsed != "390.00000000" {
		t.Errorf("fx_rate_used = %q, want the exact NUMERIC text 390.00000000", *order.FxRateUsed)
	}

	// A base-currency order involves no rate, and says so with NULL rather
	// than with a decorative 1.0.
	userID2 := seedUserWithCart(t, "usd-buyer@test.local", variantID, 1)
	usdOrder, err := s.CreateOrder(ctx, userID2, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder(USD): %v", err)
	}
	if usdOrder.FxRateUsed != nil {
		t.Errorf("fx_rate_used = %q for a USD order, want nil", *usdOrder.FxRateUsed)
	}
	// $16 of pollen, base waived — this buyer's first order too (E7).
	if usdOrder.TotalMinor != 1600 {
		t.Errorf("USD total = %d, want 1600 (first delivery free)", usdOrder.TotalMinor)
	}
}

// Browsing degrades over a market it cannot price; charging must not. The
// alternative to failing here is billing someone zero.
func TestCreateOrder_RefusesAMarketItCannotPrice(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	// A third market with no shelf prices and no rate on file — the shape of
	// a half-finished expansion, which is exactly when this bug ships.
	if _, err := testPool.Exec(ctx, `
		INSERT INTO currencies (code, symbol, minor_exponent, sort_order)
		VALUES ('EUR', '€', 2, 3)`); err != nil {
		t.Fatalf("adding EUR: %v", err)
	}
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(),
			`DELETE FROM currencies WHERE code = 'EUR'`); err != nil {
			t.Fatalf("removing EUR: %v", err)
		}
	})

	variantID := seedPricedProduct(t, "venom", "VEN-1", 10, domain.Money{domain.CurrencyUSD: 2800})
	userID := seedUserWithCart(t, "eur-buyer@test.local", variantID, 1)

	if _, ok := priceIn(t, variantID, "EUR"); ok {
		t.Fatal("the view invented a EUR price with no shelf price and no rate")
	}

	_, err := s.CreateOrder(ctx, userID, domain.View{Currency: "EUR"}, testCheckout())
	if !errors.Is(err, domain.ErrPriceUnavailable) {
		t.Fatalf("CreateOrder in EUR: err = %v, want ErrPriceUnavailable", err)
	}

	// ...and nothing was charged, reserved or cleared on the way out.
	var orders, cartLines int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM cart_items`).Scan(&cartLines); err != nil {
		t.Fatal(err)
	}
	if orders != 0 || cartLines != 1 {
		t.Errorf("after the refusal: %d orders, %d cart lines; want 0 and 1", orders, cartLines)
	}
}

func TestUpdateVariant_ReplacesThePriceSetWholesale(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "wax", "WAX-1", 5, domain.Money{
		domain.CurrencyUSD: 900, domain.CurrencyAMD: 4300,
	})

	// Dropping AMD from the map must REMOVE the shelf price, not leave the
	// old 4,300 behind — otherwise there is no way to put a variant back on
	// the converted fallback.
	if err := s.UpdateVariant(ctx, variantID, domain.Money{domain.CurrencyUSD: 1000}, 7); err != nil {
		t.Fatalf("UpdateVariant: %v", err)
	}

	var rows int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM variant_prices WHERE variant_id = $1`, variantID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Errorf("%d price rows survived, want 1", rows)
	}
	// 1000 × 3.90 = 3,900 drams, snapped to the step — the fallback, not the
	// stale shelf price.
	if got, _ := priceIn(t, variantID, domain.CurrencyAMD); got != 3900 {
		t.Errorf("AMD price = %d, want the converted 3900", got)
	}
}
