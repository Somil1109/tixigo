import { Film, Menu, Search, Ticket, UserRound, X } from "lucide-react";
import { Link, Route, Routes } from "react-router-dom";
import { useState } from "react";
import { HomePage } from "./pages/HomePage";
import { useAuth } from "./features/auth/AuthContext";
import { ForgotPasswordPage, LoginPage, RegisterPage, ResetPasswordPage, VerifyEmailPage } from "./features/auth/AuthPages";
import { AdminPage } from "./pages/AdminPage";
import { OrganiserPage } from "./pages/OrganiserPage";
import { MoviesPage } from "./pages/MoviesPage";
import { MovieDetailsPage } from "./pages/MovieDetailsPage";
import { SeatMapPage } from "./pages/SeatMapPage";
import { CheckoutPage } from "./pages/CheckoutPage";
import { BookingsPage } from "./pages/BookingsPage";
import { WaitlistPage } from "./pages/WaitlistPage";
import { TicketValidationPage } from "./pages/TicketValidationPage";

function Header() {
  const {user,logout}=useAuth();
  const [open,setOpen]=useState(false);
  const close=()=>setOpen(false);
  return <header className="site-header">
    <Link className="brand" to="/"><Ticket size={23} fill="currentColor" /> Tixigo</Link>
    <nav><Link to="/movies">Movies</Link><Link to="/about">How it works</Link></nav>
    <div className="header-actions"><button className="icon-button" aria-label="Search"><Search size={19} /></button>{user?.role==="customer"&&<><Link className="desktop-role-link" to="/bookings">My bookings</Link><Link className="desktop-role-link" to="/waitlist">Waitlist</Link></>}{user?.role==="admin"&&<Link className="desktop-role-link" to="/admin">Admin</Link>}{(user?.role==="organiser"||user?.role==="admin")&&<><Link className="desktop-role-link" to="/organiser">Organiser</Link><Link className="desktop-role-link" to="/validate-ticket">Admit</Link></>}{user?<button className="login" onClick={()=>void logout()}><UserRound size={17}/>{user.fullName} · Sign out</button>:<Link className="login" to="/login"><UserRound size={17} /> Sign in</Link>}<button className="menu" aria-label={open?"Close menu":"Open menu"} aria-expanded={open} onClick={()=>setOpen(value=>!value)}>{open?<X size={20}/>:<Menu size={20}/>}</button></div>
    {open&&<nav className="mobile-nav" aria-label="Mobile navigation"><Link onClick={close} to="/movies">Movies</Link>{user?.role==="customer"&&<><Link onClick={close} to="/bookings">My bookings</Link><Link onClick={close} to="/waitlist">Waitlist</Link></>}{user?.role==="admin"&&<Link onClick={close} to="/admin">Admin dashboard</Link>}{(user?.role==="organiser"||user?.role==="admin")&&<><Link onClick={close} to="/organiser">Organiser workspace</Link><Link onClick={close} to="/validate-ticket">Admit ticket</Link></>}{user?<button onClick={()=>{close();void logout()}}>Sign out</button>:<Link onClick={close} to="/login">Sign in</Link>}</nav>}
  </header>;
}

function Placeholder({ title }: { title: string }) {
  return <main className="placeholder"><Film size={38} /><h1>{title}</h1><p>This page is being brought to the big screen.</p></main>;
}

export function App() {
  return <><Header /><Routes>
    <Route path="/" element={<HomePage />} />
    <Route path="/movies" element={<MoviesPage />} /><Route path="/movies/:id" element={<MovieDetailsPage/>}/>
    <Route path="/screenings/:screeningId/seats" element={<SeatMapPage/>}/>
    <Route path="/checkout/:holdId" element={<CheckoutPage/>}/>
    <Route path="/bookings" element={<BookingsPage/>}/>
    <Route path="/waitlist" element={<WaitlistPage/>}/>
    <Route path="/validate-ticket" element={<TicketValidationPage/>}/>
    <Route path="/login" element={<LoginPage />} /><Route path="/register" element={<RegisterPage />} /><Route path="/forgot-password" element={<ForgotPasswordPage />} /><Route path="/reset-password" element={<ResetPasswordPage />} /><Route path="/verify-email" element={<VerifyEmailPage />} />
    <Route path="/admin" element={<AdminPage/>}/>
    <Route path="/organiser" element={<OrganiserPage/>}/>
    <Route path="*" element={<Placeholder title="Coming soon" />} />
  </Routes></>;
}
