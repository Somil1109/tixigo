package waitlist

import (
	"context"
	"fmt"
	"html"

	"github.com/tixigo/tixigo-api/internal/notification"
)

func NotifyOffers(ctx context.Context, sender notification.EmailSender, offers []Offer) {
	for _, offer := range offers {
		body := fmt.Sprintf(`<h1>Seats are ready for you</h1><p>%d seats for <strong>%s</strong> at %s are reserved until %s.</p><p>Open your Tixigo waitlist to complete checkout.</p>`, offer.Quantity, html.EscapeString(offer.MovieTitle), html.EscapeString(offer.VenueName), offer.ExpiresAt.Format("02 Jan 2006, 03:04 PM"))
		_ = sender.Send(ctx, offer.CustomerEmail, "Your Tixigo waitlist offer", body)
	}
}
