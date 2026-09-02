package domain

import (
	"errors"
	"strings"
	"time"
)

// Review statuses. Nothing reaches the storefront until a human publishes it,
// and `pending` is the zero value on purpose — forgetting to moderate fails
// CLOSED.
const (
	ReviewPending   = "pending"
	ReviewPublished = "published"
	ReviewRejected  = "rejected"
)

// ReviewStatuses is the whole set, and — like ProductSort — the only way a
// caller-supplied status reaches SQL is by matching one of these.
var ReviewStatuses = []string{ReviewPending, ReviewPublished, ReviewRejected}

func ValidReviewStatus(s string) bool {
	for _, v := range ReviewStatuses {
		if v == s {
			return true
		}
	}
	return false
}

var (
	// ErrNotPurchased is the "verified purchase" rule: you may review what
	// you have actually received. A 403, not a 404 — the product exists and
	// the reviewer is known; what is missing is the standing to speak.
	ErrNotPurchased = errors.New("no delivered order contains this product")
	// ErrAlreadyReviewed maps the UNIQUE (product_id, user_id) violation.
	// The constraint, not this check, is what makes it true under
	// concurrency: two simultaneous submissions would both pass an
	// application-level "have you reviewed this?" test.
	ErrAlreadyReviewed = errors.New("this product has already been reviewed by this user")
)

type Review struct {
	ID        int64
	ProductID int64
	UserID    int64
	Rating    int
	Title     string
	Body      string
	Status    string
	CreatedAt time.Time

	// AuthorEmail is joined in for display. The API renders a DISPLAY NAME
	// from it rather than publishing the address — see the note in the API
	// layer; a shop that prints its customers' email addresses next to their
	// opinions has invented a spam list.
	AuthorEmail string
}

// RatingSummary is the denormalized aggregate carried on every product.
type RatingSummary struct {
	Average float64
	Count   int
}

// ReviewFilter is one page of reviews. Status is empty for the public read
// (which forces `published`) and set by the admin moderation queue.
type ReviewFilter struct {
	ProductSlug string
	Status      string
	Page        int
	PerPage     int
}

func (f ReviewFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
}

// ReviewMaxTitle / ReviewMaxBody bound what a stranger can store in the
// database. The API also caps the request body, but that limit is about
// resource use; these are about the shape of a review.
const (
	ReviewMaxTitle = 120
	ReviewMaxBody  = 4000
)

// ValidateReview checks a submission. The rating is the only required field:
// a star with no words is a legitimate review, and demanding prose to leave
// one is how shops end up with fewer, angrier reviews.
func ValidateReview(r Review) map[string]string {
	fields := make(map[string]string)

	if r.Rating < 1 || r.Rating > 5 {
		fields["rating"] = ValidationRatingRange
	}
	if len([]rune(r.Title)) > ReviewMaxTitle {
		fields["title"] = ValidationTooLong
	}
	if len([]rune(r.Body)) > ReviewMaxBody {
		fields["body"] = ValidationTooLong
	}
	return fields
}

// TrimReview normalises the free text before storage. Whitespace-only input
// becomes empty rather than a validation error: the reviewer meant "no
// title", and rejecting that would be pedantry — but storing "   " would put
// a blank line in the review list forever.
//
// Counted in RUNES, not bytes, everywhere above: "Շատ լավ մեղր է" is 14
// characters and 25 bytes, and a byte limit would silently allow a third
// fewer characters in Armenian than in English.
func TrimReview(r Review) Review {
	r.Title = strings.TrimSpace(r.Title)
	r.Body = strings.TrimSpace(r.Body)
	return r
}
