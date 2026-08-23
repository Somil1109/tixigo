import { Film, Menu, Search, Ticket, UserRound } from "lucide-react";
import { Link, Route, Routes } from "react-router-dom";
import { HomePage } from "./pages/HomePage";
import { useAuth } from "./features/auth/AuthContext";
import { ForgotPasswordPage, LoginPage, RegisterPage, ResetPasswordPage, VerifyEmailPage } from "./features/auth/AuthPages";
import { AdminPage } from "./pages/AdminPage";
import { OrganiserPage } from "./pages/OrganiserPage";
import { MoviesPage } from "./pages/MoviesPage";
import { MovieDetailsPage } from "./pages/MovieDetailsPage";
import { SeatMapPage } from "./pages/SeatMapPage";

function Header() {
  const {user,logout}=useAuth();
  return <header className="site-header">
    <Link className="brand" to="/"><Ticket size={23} fill="currentColor" /> Tixigo</Link>
    <nav><Link to="/movies">Movies</Link><Link to="/about">How it works</Link></nav>
    <div className="header-actions"><button className="icon-button" aria-label="Search"><Search size={19} /></button>{user?.role==="admin"&&<Link to="/admin">Admin</Link>}{(user?.role==="organiser"||user?.role==="admin")&&<Link to="/organiser">Organiser</Link>}{user?<button className="login" onClick={()=>void logout()}><UserRound size={17}/>{user.fullName} · Sign out</button>:<Link className="login" to="/login"><UserRound size={17} /> Sign in</Link>}<button className="menu" aria-label="Menu"><Menu size={20} /></button></div>
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
    <Route path="/login" element={<LoginPage />} /><Route path="/register" element={<RegisterPage />} /><Route path="/forgot-password" element={<ForgotPasswordPage />} /><Route path="/reset-password" element={<ResetPasswordPage />} /><Route path="/verify-email" element={<VerifyEmailPage />} />
    <Route path="/admin" element={<AdminPage/>}/>
    <Route path="/organiser" element={<OrganiserPage/>}/>
    <Route path="*" element={<Placeholder title="Coming soon" />} />
  </Routes></>;
}
