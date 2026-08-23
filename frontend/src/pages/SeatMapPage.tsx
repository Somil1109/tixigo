import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useAuth } from "../features/auth/AuthContext";
import { api, ApiError } from "../lib/api";
import { config } from "../lib/config";

type Seat = { id: string; key: string; row: string; number: string; category: string; pricePaise: number; status: "available" | "held" | "booked" | "waitlist_reserved" };
type MapData = { movieTitle: string; venueName: string; startsAt: string; seats: Seat[] };
type Hold = { id: string; expiresAt: string; seats: Seat[] };

export function SeatMapPage() {
  const { screeningId } = useParams();
  const navigate = useNavigate();
  const auth = useAuth();
  const [data, setData] = useState<MapData | null>(null);
  const [selected, setSelected] = useState<string[]>([]);
  const [hold, setHold] = useState<Hold | null>(null);
  const [remaining, setRemaining] = useState(0);
  const [live, setLive] = useState(false);
  const [error, setError] = useState("");
  const [waitlistCategory, setWaitlistCategory] = useState("");
  const [waitlistQuantity, setWaitlistQuantity] = useState(1);
  const [waitlistMessage, setWaitlistMessage] = useState("");

  const refreshMap = useCallback(async () => {
    if (!screeningId) return;
    const result = await api<{ data: MapData }>(`/screenings/${screeningId}/seats`);
    setData(result.data);
    const available = new Set(result.data.seats.filter(seat => seat.status === "available").map(seat => seat.id));
    setSelected(current => current.filter(id => available.has(id)));
  }, [screeningId]);

  useEffect(() => { void refreshMap().catch(() => setError("Seat map could not be loaded.")); }, [refreshMap]);

  useEffect(() => {
    if (!screeningId) return;
    let socket: WebSocket | null = null;
    let reconnectTimer: number | undefined;
    let attempt = 0;
    let stopped = false;

    const connect = () => {
      socket = new WebSocket(`${config.webSocketBaseUrl}/screenings/${screeningId}`);
      socket.onopen = () => { attempt = 0; setLive(true); };
      socket.onmessage = event => {
        try {
          const message = JSON.parse(event.data) as { type?: string; screeningId?: string };
          if (message.type === "seats.updated" && message.screeningId === screeningId) void refreshMap();
        } catch { /* Ignore malformed messages and keep the live connection open. */ }
      };
      socket.onclose = () => {
        setLive(false);
        if (!stopped) reconnectTimer = window.setTimeout(connect, Math.min(1000 * 2 ** attempt++, 10_000));
      };
      socket.onerror = () => socket?.close();
    };

    connect();
    return () => {
      stopped = true;
      setLive(false);
      if (reconnectTimer) window.clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [refreshMap, screeningId]);

  useEffect(() => {
    if (!hold) return;
    const update = () => {
      const seconds = Math.max(0, Math.ceil((new Date(hold.expiresAt).getTime() - Date.now()) / 1000));
      setRemaining(seconds);
      if (seconds === 0) {
        setHold(null);
        setSelected([]);
        setError("Your seat hold expired. Please select seats again.");
      }
    };
    update();
    const timer = window.setInterval(update, 1000);
    return () => window.clearInterval(timer);
  }, [hold]);

  const rows = useMemo(() => {
    const grouped = new Map<string, Seat[]>();
    for (const seat of data?.seats ?? []) grouped.set(seat.row, [...(grouped.get(seat.row) ?? []), seat]);
    return [...grouped.entries()];
  }, [data]);
  const categories = useMemo(() => [...new Set(data?.seats.map(seat => seat.category) ?? [])], [data]);

  if (!data) return <main className="placeholder"><p>Loading seat map…</p></main>;
  const chosen = hold?.seats ?? data.seats.filter(seat => selected.includes(seat.id));

  function toggle(seat: Seat) {
    if (hold || seat.status !== "available") return;
    setSelected(current => current.includes(seat.id) ? current.filter(id => id !== seat.id) : current.length < 10 ? [...current, seat.id] : current);
  }

  async function createHold() {
    if (!auth.user) { navigate("/login"); return; }
    try {
      const result = await auth.request<{ data: Hold }>(`/screenings/${screeningId}/holds`, { method: "POST", body: JSON.stringify({ seatIds: selected }) });
      setHold(result.data);
      setError("");
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "Seats could not be held. Refresh and try again.");
      await refreshMap();
    }
  }

  async function release() {
    if (!hold) return;
    try {
      await auth.request(`/holds/${hold.id}`, { method: "DELETE" });
      setHold(null);
      setSelected([]);
      await refreshMap();
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "Seat hold could not be released.");
    }
  }

  async function joinWaitlist() {
    if (!auth.user) { navigate("/login"); return; }
    const category = waitlistCategory || categories[0];
    if (!category) return;
    try {
      await auth.request(`/screenings/${screeningId}/waitlist`, { method: "POST", body: JSON.stringify({ category, quantity: waitlistQuantity }) });
      setWaitlistMessage("You're on the waitlist. We'll email you when seats are reserved.");
    } catch (reason) {
      setWaitlistMessage(reason instanceof ApiError ? reason.message : "Could not join the waitlist.");
    }
  }

  const clock = `${String(Math.floor(remaining / 60)).padStart(2, "0")}:${String(remaining % 60).padStart(2, "0")}`;
  return <main className="seat-page">
    <header><span className="eyebrow">CHOOSE YOUR SEATS</span><h1>{data.movieTitle}</h1><p>{data.venueName} · {new Date(data.startsAt).toLocaleString("en-IN")}</p><span className={`live-status ${live ? "connected" : ""}`}><i/>{live ? "Live seat availability" : "Reconnecting live updates…"}</span>{hold && <strong className="hold-clock">Held for {clock}</strong>}</header>
    <div className="screen">SCREEN</div>
    <section className="seat-map">{rows.map(([row, seats]) => <div className="seat-row" key={row}><strong>{row}</strong><div>{seats.map(seat => <button key={seat.id} className={`seat ${seat.status} ${(selected.includes(seat.id) || hold?.seats.some(item => item.id === seat.id)) ? "selected" : ""}`} disabled={seat.status !== "available" || !!hold} onClick={() => toggle(seat)} title={`${seat.key} · ${seat.category} · ₹${seat.pricePaise / 100}`}>{seat.number}</button>)}</div><strong>{row}</strong></div>)}</section>
    <div className="seat-legend"><span><i className="available"/>Available</span><span><i className="selected"/>Selected</span><span><i className="held"/>Held</span><span><i className="booked"/>Booked</span></div>
    <section className="waitlist-join"><div><span className="eyebrow">CAN'T FIND ENOUGH SEATS?</span><h2>Join the waitlist</h2><p>Choose a category and quantity. Any matching seats can fulfil your request.</p></div><select value={waitlistCategory || categories[0] || ""} onChange={event => setWaitlistCategory(event.target.value)}>{categories.map(category => <option value={category} key={category}>{category}</option>)}</select><input aria-label="Waitlist quantity" type="number" min="1" max="10" value={waitlistQuantity} onChange={event => setWaitlistQuantity(Number(event.target.value))}/><button onClick={() => void joinWaitlist()}>Join waitlist</button>{waitlistMessage && <small>{waitlistMessage}</small>}</section>
    {error && <p className="form-error seat-error">{error}</p>}
    <aside className="seat-summary"><div><strong>{chosen.length} seat{chosen.length === 1 ? "" : "s"}</strong><span>{chosen.map(seat => seat.key).join(", ") || "Select up to 10 seats"}</span></div><div><strong>₹{(chosen.reduce((sum, seat) => sum + seat.pricePaise, 0) / 100).toFixed(0)}</strong>{hold ? <><button className="secondary" onClick={() => void release()}>Release</button><button onClick={() => navigate(`/checkout/${hold.id}`)}>Checkout</button></> : <button disabled={!chosen.length} onClick={() => void createHold()}>Hold seats</button>}</div></aside>
  </main>;
}
