import { expect, test } from "@playwright/test";

test("customer holds seats and completes simulated checkout", async ({ page }) => {
  const startsAt = new Date(Date.now() + 86_400_000).toISOString();
  const seats = [{ id: "seat-1", key: "A1", row: "A", number: "1", category: "standard", pricePaise: 25000, status: "available" }];
  await page.route("**/api/v1/**", async route => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const json = (value: unknown, status = 200) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(value) });
    if (path.endsWith("/auth/refresh")) return json({ data: { accessToken: "test-token", user: { id: "user-1", email: "customer@example.com", fullName: "Customer", role: "customer", emailVerifiedAt: new Date().toISOString() } } });
    if (path.endsWith("/screenings/show-1/seats")) return json({ data: { movieTitle: "Mumbai Nights", venueName: "Regal Cinema", startsAt, seats } });
    if (path.endsWith("/screenings/show-1/holds")) return json({ data: { id: "hold-1", expiresAt: new Date(Date.now() + 600_000).toISOString(), seats } }, 201);
    if (path.endsWith("/holds/hold-1") && request.method() === "GET") return json({ data: { id: "hold-1", movieTitle: "Mumbai Nights", venueName: "Regal Cinema", startsAt, totalPaise: 25000, expiresAt: new Date(Date.now() + 600_000).toISOString(), seats } });
    if (path.endsWith("/holds/hold-1/checkout")) return json({ data: { booking: { id: "booking-1", reference: "TIX-TEST123", movieTitle: "Mumbai Nights", venueName: "Regal Cinema", startsAt, seats, totalPaise: 25000 }, qrCode: "data:image/png;base64,iVBORw0KGgo=", emailSent: true } }, 201);
    return json({ data: [] });
  });

  await page.goto("/screenings/show-1/seats");
  await page.getByRole("button", { name: "1" }).click();
  await page.getByRole("button", { name: "Hold seats" }).click();
  await page.getByRole("button", { name: "Checkout" }).click();
  await expect(page.getByRole("heading", { name: "Confirm your booking" })).toBeVisible();
  await page.getByRole("button", { name: "Confirm payment" }).click();
  await expect(page.getByText("BOOKING CONFIRMED")).toBeVisible();
  await expect(page.getByText("TIX-TEST123")).toBeVisible();
});
