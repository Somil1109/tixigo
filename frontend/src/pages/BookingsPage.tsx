import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { CalendarDays, MapPin, Ticket, X } from "lucide-react";
import { useAuth } from "../features/auth/AuthContext";
import { ApiError } from "../lib/api";

type Seat = { id: string; key: string; pricePaise: number };
type Booking = {
  id: string;
  reference: string;
  screeningId: string;
  status: "confirmed" | "cancelled";
  movieTitle: string;
  venueName: string;
  startsAt: string;
  totalPaise: number;
  seats: Seat[];
  cancelledAt?: string;
  canCancel: boolean;
};

export function BookingsPage() {
  const auth = useAuth();
  const [items, setItems] = useState<Booking[]>([]);
  const [ticket, setTicket] = useState<{ booking: Booking; qrCode: string } | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState("");

  async function load() {
    if (!auth.user) return;
    try {
      const result = await auth.request<{ data: Booking[] }>("/bookings");
      setItems(result.data);
      setError("");
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "Bookings could not be loaded.");
    }
  }

  useEffect(() => { void load(); }, [auth.user]);

  async function showTicket(id: string) {
    setBusy(id);
    try {
      const result = await auth.request<{ data: { booking: Booking; qrCode: string } }>(`/bookings/${id}`);
      setTicket(result.data);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "Ticket could not be loaded.");
    } finally {
      setBusy(null);
    }
  }

  async function cancel(item: Booking) {
    if (!window.confirm(`Cancel booking ${item.reference}? All seats in this booking will be released.`)) return;
    setBusy(item.id);
    try {
      const result = await auth.request<{ data: { booking: Booking } }>(`/bookings/${item.id}/cancel`, { method: "POST" });
      setItems(current => current.map(existing => existing.id === item.id ? result.data.booking : existing));
      setTicket(current => current?.booking.id === item.id ? { ...current, booking: result.data.booking } : current);
      setError("");
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "Booking could not be cancelled.");
    } finally {
      setBusy(null);
    }
  }

  if (auth.loading) return <main className="placeholder"><p>Loading bookings…</p></main>;
  if (!auth.user) return <main className="placeholder"><Ticket size={38}/><h1>Your tickets</h1><p>Sign in to view your bookings.</p><Link className="auth-primary-link" to="/login">Sign in</Link></main>;

  return <main className="bookings-page">
    <header><span className="eyebrow">MY TIXIGO</span><h1>Bookings & tickets</h1><p>Every booking, showtime, and QR ticket in one place.</p></header>
    {error && <p className="form-error bookings-error">{error}</p>}
    {!items.length && !error ? <section className="empty-bookings"><Ticket size={34}/><h2>No bookings yet</h2><p>Your first cinema night is waiting.</p><Link className="auth-primary-link" to="/movies">Browse movies</Link></section> :
      <section className="booking-list">{items.map(item => <article className={`booking-item ${item.status}`} key={item.id}>
        <div className="booking-main"><span className="booking-status">{item.status}</span><h2>{item.movieTitle}</h2><strong>{item.reference}</strong></div>
        <div className="booking-meta"><span><CalendarDays size={16}/>{new Date(item.startsAt).toLocaleString("en-IN")}</span><span><MapPin size={16}/>{item.venueName}</span><span><Ticket size={16}/>Seats {item.seats.map(seat => seat.key).join(", ")}</span></div>
        <div className="booking-actions"><strong>₹{(item.totalPaise / 100).toFixed(0)}</strong><button disabled={busy === item.id} onClick={() => void showTicket(item.id)}>{busy === item.id ? "Loading…" : "View ticket"}</button>{item.canCancel && <button className="danger" disabled={busy === item.id} onClick={() => void cancel(item)}>Cancel booking</button>}</div>
      </article>)}</section>}
    {ticket && <div className="ticket-overlay" role="dialog" aria-modal="true" aria-label="Booking ticket"><section className="ticket-card"><button className="ticket-close" aria-label="Close ticket" onClick={() => setTicket(null)}><X/></button><span className="eyebrow">{ticket.booking.status === "confirmed" ? "VALID TICKET" : "CANCELLED"}</span><h1>{ticket.booking.movieTitle}</h1><img src={ticket.qrCode} alt="Booking QR ticket"/><strong>{ticket.booking.reference}</strong><p>{ticket.booking.venueName} · {new Date(ticket.booking.startsAt).toLocaleString("en-IN")}</p><p>Seats {ticket.booking.seats.map(seat => seat.key).join(", ")}</p></section></div>}
  </main>;
}
