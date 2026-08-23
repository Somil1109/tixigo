import { useCallback, useEffect, useState, type FormEvent } from "react";
import { useAuth, type User } from "../features/auth/AuthContext";
import { ApiError } from "../lib/api";

type Venue = { id: string; name: string };
type Pending = { id: string; title: string; language: string; ageRating: string; durationMinutes: number };
const sample = JSON.stringify({ categories: [{ key: "standard", label: "Standard" }, { key: "premium", label: "Premium" }], rows: [{ label: "A", seats: [{ number: "1", category: "premium", column: 1 }, { number: "2", category: "premium", column: 2 }] }] }, null, 2);

export function AdminPage() {
  const auth = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [venues, setVenues] = useState<Venue[]>([]);
  const [pending, setPending] = useState<Pending[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState("");

  const load = useCallback(async () => {
    if (auth.user?.role !== "admin") return;
    try {
      const [userResult, venueResult, pendingResult] = await Promise.all([auth.request<{ data: User[] }>("/admin/users"), auth.request<{ data: Venue[] }>("/admin/venues"), auth.request<{ data: Pending[] }>("/admin/movies/pending")]);
      setUsers(userResult.data); setVenues(venueResult.data); setPending(pendingResult.data);
    } catch (reason) {
      setStatus(reason instanceof ApiError ? reason.message : "Dashboard data could not be loaded.");
    } finally { setLoading(false); }
  }, [auth]);
  useEffect(() => { void load(); }, [load]);

  async function run(action: () => Promise<void>, success: string) {
    if (busy) return;
    setBusy(true); setStatus("");
    try { await action(); setStatus(success); await load(); }
    catch (reason) { setStatus(reason instanceof ApiError ? reason.message : "The action could not be completed."); }
    finally { setBusy(false); }
  }

  if (auth.user?.role !== "admin") return <main className="placeholder"><h1>Admin access required</h1></main>;
  if (loading) return <main className="placeholder"><p>Loading admin dashboard…</p></main>;

  async function changeRole(id: string, role: "customer" | "organiser") { await run(() => auth.request(`/admin/users/${id}/role`, { method: "PATCH", body: JSON.stringify({ role }) }), "User role updated."); }
  async function createVenue(event: FormEvent<HTMLFormElement>) { event.preventDefault(); const form = new FormData(event.currentTarget); await run(() => auth.request("/admin/venues", { method: "POST", body: JSON.stringify({ name: form.get("name"), address: form.get("address"), city: form.get("city"), layout: JSON.parse(String(form.get("layout"))) }) }), "Venue created."); }
  async function review(id: string, decision: "approve" | "reject") { const reason = decision === "reject" ? window.prompt("Rejection reason") ?? "" : ""; if (decision === "reject" && !reason) return; await run(() => auth.request(`/admin/movies/${id}/review`, { method: "PATCH", body: JSON.stringify({ decision, reason }) }), decision === "approve" ? "Movie approved." : "Movie rejected."); }

  return <main className="dashboard" aria-busy={busy}><header><span className="eyebrow">ADMIN</span><h1>Operations dashboard</h1>{status && <p className="dashboard-status">{status}</p>}</header><section className="dashboard-panel"><h2>Pending movies</h2>{pending.map(movie => <div className="dashboard-row" key={movie.id}><span><strong>{movie.title}</strong><small>{movie.language} · {movie.durationMinutes} min · {movie.ageRating}</small></span><span><button disabled={busy} onClick={() => void review(movie.id, "approve")}>Approve</button><button disabled={busy} className="secondary" onClick={() => void review(movie.id, "reject")}>Reject</button></span></div>)}{!pending.length && <p className="empty-copy">No movies awaiting review.</p>}</section><section className="dashboard-grid"><div className="dashboard-panel"><h2>Users</h2>{users.map(user => <div className="dashboard-row" key={user.id}><span><strong>{user.fullName}</strong><small>{user.email} · {user.role}</small></span>{user.role !== "admin" && <button disabled={busy} onClick={() => void changeRole(user.id, user.role === "organiser" ? "customer" : "organiser")}>{user.role === "organiser" ? "Make customer" : "Promote"}</button>}</div>)}{!users.length && <p className="empty-copy">No users found.</p>}</div><div className="dashboard-panel"><h2>Create venue</h2><form className="auth-form" onSubmit={createVenue}><label>Name<input name="name" required/></label><label>Address<input name="address" required/></label><label>City<input name="city" required/></label><label>Seat layout JSON<textarea name="layout" defaultValue={sample} required/></label><button disabled={busy}>{busy ? "Saving…" : "Create venue"}</button></form><p className="empty-copy">{venues.length} venue{venues.length === 1 ? "" : "s"} configured</p></div></section></main>;
}
