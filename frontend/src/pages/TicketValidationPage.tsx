import { useState, type FormEvent } from "react";
import { CheckCircle2, ScanLine, XCircle } from "lucide-react";
import { useAuth } from "../features/auth/AuthContext";
import { ApiError } from "../lib/api";

type Admission = { bookingId: string; reference: string; movieTitle: string; venueName: string; startsAt: string; seats: string[]; checkedInAt: string };

export function TicketValidationPage() {
  const auth = useAuth();
  const [result, setResult] = useState<Admission | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  if (!auth.user || (auth.user.role !== "organiser" && auth.user.role !== "admin")) return <main className="placeholder"><ScanLine size={38}/><h1>Staff access required</h1></main>;

  async function validate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setBusy(true); setResult(null); setError("");
    const reference = String(new FormData(event.currentTarget).get("reference"));
    try {
      const response = await auth.request<{ data: Admission }>("/tickets/admit", { method: "POST", body: JSON.stringify({ reference }) });
      setResult(response.data); event.currentTarget.reset();
    } catch (reason) {
      setError(reason instanceof ApiError ? reason.message : "Ticket could not be validated.");
    } finally { setBusy(false); }
  }

  return <main className="admission-page"><section className="admission-card"><ScanLine size={42}/><span className="eyebrow">STAFF ADMISSION</span><h1>Validate a ticket</h1><p>Scan the QR with any device scanner and paste its TIX reference, or enter the reference printed below the QR.</p><form className="auth-form" onSubmit={validate}><label>Ticket reference<input name="reference" pattern="TIX-[A-Za-z0-9]+" placeholder="TIX-XXXXXXXXXXXX" autoComplete="off" autoFocus required/></label><button disabled={busy}>{busy ? "Checking…" : "Validate and admit"}</button></form>{result && <div className="admission-result valid"><CheckCircle2/><div><strong>Admission approved</strong><span>{result.movieTitle} · {result.venueName}</span><span>{new Date(result.startsAt).toLocaleString("en-IN")} · Seats {result.seats.join(", ")}</span><small>{result.reference} marked used at {new Date(result.checkedInAt).toLocaleTimeString("en-IN")}</small></div></div>}{error && <div className="admission-result invalid"><XCircle/><div><strong>Admission denied</strong><span>{error}</span></div></div>}</section></main>;
}
