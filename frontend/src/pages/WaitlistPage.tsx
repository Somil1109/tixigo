import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Clock3, ListPlus, MapPin } from "lucide-react";
import { useAuth } from "../features/auth/AuthContext";
import { ApiError } from "../lib/api";

type Entry = { id: string; screeningId: string; movieTitle: string; venueName: string; startsAt: string; category: string; quantity: number; status: string; offerExpiresAt?: string; holdId?: string };

export function WaitlistPage() {
  const auth = useAuth();
  const [items, setItems] = useState<Entry[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);

  async function load() {
    if (!auth.user) return;
    try {
      const result = await auth.request<{ data: Entry[] }>("/waitlist");
      setItems(result.data);
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "Waitlist entries could not be loaded.");
    } finally { setLoading(false); }
  }

  useEffect(() => { void load(); }, [auth.user]);

  async function cancel(id: string) {
    if (busy) return; setBusy(id); setError("");
    try { await auth.request(`/waitlist/${id}`, { method: "DELETE" }); await load(); }
    catch (reason) { setError(reason instanceof ApiError ? reason.message : "Waitlist entry could not be removed."); }
    finally { setBusy(null); }
  }

  if (!auth.user) return <main className="placeholder"><ListPlus size={38}/><h1>Your waitlist</h1><p>Sign in to manage your waitlist.</p><Link className="auth-primary-link" to="/login">Sign in</Link></main>;
  if(loading)return <main className="placeholder"><p>Loading your waitlist…</p></main>;
  return <main className="bookings-page"><header><span className="eyebrow">SEAT ALERTS</span><h1>Your waitlist</h1><p>When matching seats open up, they are reserved for you for one hour.</p></header>{error && <p className="form-error bookings-error">{error}</p>}<section className="booking-list">{items.map(item => <article className={`booking-item ${item.status}`} key={item.id}><div className="booking-main"><span className="booking-status">{item.status}</span><h2>{item.movieTitle}</h2><strong>{item.quantity} × {item.category}</strong></div><div className="booking-meta"><span><Clock3 size={16}/>{new Date(item.startsAt).toLocaleString("en-IN")}</span><span><MapPin size={16}/>{item.venueName}</span>{item.offerExpiresAt && <span>Offer expires {new Date(item.offerExpiresAt).toLocaleString("en-IN")}</span>}</div><div className="booking-actions">{item.status === "offered" && item.holdId && <Link className="waitlist-checkout" to={`/checkout/${item.holdId}`}>Checkout offer</Link>}{(item.status === "waiting" || item.status === "offered") && <button disabled={busy===item.id} className="danger" onClick={() => void cancel(item.id)}>{busy===item.id?"Leaving…":"Leave waitlist"}</button>}</div></article>)}</section>{!items.length && !error && <section className="empty-bookings"><ListPlus size={34}/><h2>No waitlist entries</h2><p>Join from a screening's seat map when your preferred seats are unavailable.</p><Link className="auth-primary-link" to="/movies">Browse movies</Link></section>}</main>;
}
