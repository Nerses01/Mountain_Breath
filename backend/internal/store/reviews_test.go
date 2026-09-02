package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// deliverOrderFor gives a user a DELIVERED order containing the product, so
// they have standing to review it. Written directly rather than through
// CreateOrder + three status transitions: this is fixture, not the thing
// under test.
func deliverOrderFor(t *testing.T, userID, variantID int64) {
	t.Helper()
	ctx := context.Background()
	var orderID int64
	if err := testPool.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_minor, subtotal_minor, currency, payment_method)
		VALUES ($1, 'delivered', 1000, 1000, 'USD', 'card') RETURNING id`, userID).Scan(&orderID); err != nil {
		t.Fatalf("seeding order: %v", err)
	}
	if _, err := testPool.Exec(ctx, `
		INSERT INTO order_items (order_id, variant_id, name_snapshot, label_snapshot,
		                         price_minor_snapshot, qty)
		VALUES ($1, $2, 'x', 'y', 1000, 1)`, orderID, variantID); err != nil {
		t.Fatalf("seeding order item: %v", err)
	}
}

func seedUser(t *testing.T, email string) int64 {
	t.Helper()
	var id int64
	if err := testPool.QueryRow(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ($1, 'x') RETURNING id`,
		email).Scan(&id); err != nil {
		t.Fatalf("seeding user %s: %v", email, err)
	}
	return id
}

func ratingOf(t *testing.T, slug string) (float64, int) {
	t.Helper()
	var avg float64
	var count int
	if err := testPool.QueryRow(context.Background(),
		`SELECT rating_avg::float8, rating_count FROM products WHERE slug = $1`,
		slug).Scan(&avg, &count); err != nil {
		t.Fatalf("reading rating: %v", err)
	}
	return avg, count
}

func variantOf(t *testing.T, slug string) int64 {
	t.Helper()
	var id int64
	if err := testPool.QueryRow(context.Background(),
		`SELECT v.id FROM product_variants v
		 JOIN products p ON p.id = v.product_id
		 WHERE p.slug = $1 ORDER BY v.id LIMIT 1`, slug).Scan(&id); err != nil {
		t.Fatalf("finding a variant of %s: %v", slug, err)
	}
	return id
}

// The phase's central claim: the aggregate agrees with the rows, at every
// step of publish → change → reject. It is recomputed rather than nudged, so
// this holds no matter what order the moderator works in.
func TestReviewAggregate_StaysHonestThroughModeration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	variant := variantOf(t, "honey")
	alice := seedUser(t, "alice@test.local")
	bob := seedUser(t, "bob@test.local")
	deliverOrderFor(t, alice, variant)
	deliverOrderFor(t, bob, variant)

	aliceReview := domain.Review{UserID: alice, Rating: 5, Body: "Excellent"}
	if err := s.CreateReview(ctx, &aliceReview, "honey"); err != nil {
		t.Fatalf("CreateReview: %v", err)
	}

	// A new review is PENDING, so it must not move the public average — the
	// whole reason moderation exists.
	if avg, n := ratingOf(t, "honey"); avg != 0 || n != 0 {
		t.Errorf("pending review moved the average: %.2f (%d)", avg, n)
	}
	if aliceReview.Status != domain.ReviewPending {
		t.Errorf("status = %q, want pending", aliceReview.Status)
	}

	if _, err := s.UpdateReviewStatus(ctx, aliceReview.ID, domain.ReviewPublished); err != nil {
		t.Fatalf("publishing: %v", err)
	}
	if avg, n := ratingOf(t, "honey"); avg != 5 || n != 1 {
		t.Errorf("after publish: %.2f (%d), want 5.00 (1)", avg, n)
	}

	bobReview := domain.Review{UserID: bob, Rating: 4, Body: "Good"}
	if err := s.CreateReview(ctx, &bobReview, "honey"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateReviewStatus(ctx, bobReview.ID, domain.ReviewPublished); err != nil {
		t.Fatal(err)
	}
	if avg, n := ratingOf(t, "honey"); avg != 4.5 || n != 2 {
		t.Errorf("after two: %.2f (%d), want 4.50 (2)", avg, n)
	}

	// Rejecting a published review takes it back OUT of the average. An
	// incremental implementation is where this usually breaks.
	if _, err := s.UpdateReviewStatus(ctx, aliceReview.ID, domain.ReviewRejected); err != nil {
		t.Fatal(err)
	}
	if avg, n := ratingOf(t, "honey"); avg != 4 || n != 1 {
		t.Errorf("after reject: %.2f (%d), want 4.00 (1)", avg, n)
	}

	// ...and back in again. Recomputation makes the operation idempotent and
	// order-independent, which nudging a stored total never is.
	if _, err := s.UpdateReviewStatus(ctx, aliceReview.ID, domain.ReviewPublished); err != nil {
		t.Fatal(err)
	}
	if avg, n := ratingOf(t, "honey"); avg != 4.5 || n != 2 {
		t.Errorf("after re-publish: %.2f (%d), want 4.50 (2)", avg, n)
	}

	// Rejecting everything must return the product to zero, not to NULL —
	// avg() over no rows is NULL and the column is NOT NULL.
	for _, id := range []int64{aliceReview.ID, bobReview.ID} {
		if _, err := s.UpdateReviewStatus(ctx, id, domain.ReviewRejected); err != nil {
			t.Fatal(err)
		}
	}
	if avg, n := ratingOf(t, "honey"); avg != 0 || n != 0 {
		t.Errorf("after rejecting all: %.2f (%d), want 0.00 (0)", avg, n)
	}
}

// A rating that does not divide evenly is where a stored average has to be
// honest about its precision.
func TestReviewAggregate_RoundsToTwoDecimals(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	variant := variantOf(t, "honey")
	// 5 + 4 + 4 = 13 / 3 = 4.333…
	for i, rating := range []int{5, 4, 4} {
		user := seedUser(t, string(rune('a'+i))+"@test.local")
		deliverOrderFor(t, user, variant)
		r := domain.Review{UserID: user, Rating: rating}
		if err := s.CreateReview(ctx, &r, "honey"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.UpdateReviewStatus(ctx, r.ID, domain.ReviewPublished); err != nil {
			t.Fatal(err)
		}
	}

	avg, n := ratingOf(t, "honey")
	if n != 3 || avg != 4.33 {
		t.Errorf("average = %.4f (%d), want 4.33 (3)", avg, n)
	}
}

func TestCreateReview_RequiresADeliveredPurchase(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	stranger := seedUser(t, "stranger@test.local")
	r := domain.Review{UserID: stranger, Rating: 5, Body: "Never bought it"}
	if err := s.CreateReview(ctx, &r, "honey"); !errors.Is(err, domain.ErrNotPurchased) {
		t.Fatalf("err = %v, want ErrNotPurchased", err)
	}

	// An order that exists but has NOT been delivered is not standing
	// either — the rule is "received", not "paid".
	variant := variantOf(t, "honey")
	var orderID int64
	if err := testPool.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_minor, subtotal_minor, currency, payment_method)
		VALUES ($1, 'pending', 1000, 1000, 'USD', 'card') RETURNING id`, stranger).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx, `
		INSERT INTO order_items (order_id, variant_id, name_snapshot, label_snapshot,
		                         price_minor_snapshot, qty)
		VALUES ($1, $2, 'x', 'y', 1000, 1)`, orderID, variant); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateReview(ctx, &r, "honey"); !errors.Is(err, domain.ErrNotPurchased) {
		t.Errorf("a pending order granted standing: %v", err)
	}

	// Delivering it does.
	if _, err := testPool.Exec(ctx,
		`UPDATE orders SET status = 'delivered' WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateReview(ctx, &r, "honey"); err != nil {
		t.Errorf("a delivered order should grant standing: %v", err)
	}

	// Buying ONE product does not grant standing to review another.
	other := domain.Review{UserID: stranger, Rating: 1, Body: "Different product"}
	if err := s.CreateReview(ctx, &other, "wax"); !errors.Is(err, domain.ErrNotPurchased) {
		t.Errorf("standing leaked across products: %v", err)
	}
}

func TestCreateReview_OneReviewPerPersonPerProduct(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	variant := variantOf(t, "honey")
	user := seedUser(t, "twice@test.local")
	deliverOrderFor(t, user, variant)

	first := domain.Review{UserID: user, Rating: 5}
	if err := s.CreateReview(ctx, &first, "honey"); err != nil {
		t.Fatal(err)
	}

	// The UNIQUE constraint is what makes this true under concurrency — an
	// application-level "have you reviewed this?" check would let two
	// simultaneous submissions both through.
	second := domain.Review{UserID: user, Rating: 1}
	if err := s.CreateReview(ctx, &second, "honey"); !errors.Is(err, domain.ErrAlreadyReviewed) {
		t.Fatalf("err = %v, want ErrAlreadyReviewed", err)
	}
}

func TestCanReview(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	variant := variantOf(t, "honey")
	user := seedUser(t, "eligible@test.local")

	can, err := s.CanReview(ctx, user, "honey")
	if err != nil {
		t.Fatal(err)
	}
	if can {
		t.Error("a non-purchaser was told they could review")
	}

	deliverOrderFor(t, user, variant)
	if can, _ = s.CanReview(ctx, user, "honey"); !can {
		t.Error("a purchaser was told they could not review")
	}

	// Having already written one closes the door too, so the UI never shows
	// a form the write path would refuse with a 409.
	r := domain.Review{UserID: user, Rating: 4}
	if err := s.CreateReview(ctx, &r, "honey"); err != nil {
		t.Fatal(err)
	}
	if can, _ = s.CanReview(ctx, user, "honey"); can {
		t.Error("an existing reviewer was offered the form again")
	}
}

func TestListReviews_PublicSeesOnlyPublished(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	variant := variantOf(t, "honey")
	for i, status := range []string{domain.ReviewPublished, domain.ReviewPending, domain.ReviewRejected} {
		user := seedUser(t, string(rune('m'+i))+"@test.local")
		deliverOrderFor(t, user, variant)
		r := domain.Review{UserID: user, Rating: 5, Body: status}
		if err := s.CreateReview(ctx, &r, "honey"); err != nil {
			t.Fatal(err)
		}
		if status != domain.ReviewPending {
			if _, err := s.UpdateReviewStatus(ctx, r.ID, status); err != nil {
				t.Fatal(err)
			}
		}
	}

	published, total, err := s.ListReviews(ctx, domain.ReviewFilter{
		ProductSlug: "honey", Status: domain.ReviewPublished, Page: 1, PerPage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(published) != 1 || published[0].Body != domain.ReviewPublished {
		t.Errorf("public list = %d rows (total %d), want only the published one", len(published), total)
	}
	// The author's email comes back for the API to turn into a display name.
	if published[0].AuthorEmail == "" {
		t.Error("author email was not joined in")
	}

	// The moderation queue, filtering by status, sees the others.
	pending, total, err := s.ListReviews(ctx, domain.ReviewFilter{
		Status: domain.ReviewPending, Page: 1, PerPage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(pending) != 1 {
		t.Errorf("pending queue = %d rows (total %d), want 1", len(pending), total)
	}

	// An empty status means "every status" — the queue's default view.
	all, total, err := s.ListReviews(ctx, domain.ReviewFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(all) != 3 {
		t.Errorf("unfiltered = %d rows (total %d), want 3", len(all), total)
	}
}

// Sorting by rating needs the aggregate on the LIST query, which is the
// entire reason it is a stored column.
func TestListProducts_SortByRating(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	publish := func(slug string, email string, rating int) {
		t.Helper()
		user := seedUser(t, email)
		deliverOrderFor(t, user, variantOf(t, slug))
		r := domain.Review{UserID: user, Rating: rating}
		if err := s.CreateReview(ctx, &r, slug); err != nil {
			t.Fatal(err)
		}
		if _, err := s.UpdateReviewStatus(ctx, r.ID, domain.ReviewPublished); err != nil {
			t.Fatal(err)
		}
	}

	publish("wax", "w1@test.local", 5)
	publish("honey", "h1@test.local", 4)
	publish("honey", "h2@test.local", 4)

	got, _, err := s.ListProducts(ctx, domain.ProductFilter{
		Page: 1, PerPage: 20, Sort: domain.SortRating,
	})
	if err != nil {
		t.Fatal(err)
	}
	// wax 5.00 (1) beats honey 4.00 (2) on average; the products with no
	// reviews at all fall to the bottom.
	if slugs := slugsOf(got); slugs[0] != "wax" || slugs[1] != "honey" {
		t.Errorf("rating order = %v, want wax then honey", slugs)
	}

	// The aggregate rides on the LIST payload, not just the detail — the
	// card draws stars too.
	for _, p := range got {
		if p.Slug == "honey" && (p.Rating.Average != 4 || p.Rating.Count != 2) {
			t.Errorf("honey on the listing = %.2f (%d)", p.Rating.Average, p.Rating.Count)
		}
	}
}
