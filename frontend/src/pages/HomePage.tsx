import { ArrowRight, CalendarDays, ChevronRight, MapPin, Play, Star } from "lucide-react";

const movies = [
  { title: "The Last Horizon", genre: "Adventure · 2h 18m", rating: "8.7", art: "horizon" },
  { title: "Midnight in Mumbai", genre: "Drama · 2h 04m", rating: "8.4", art: "mumbai" },
  { title: "The Sound of Rain", genre: "Romance · 1h 56m", rating: "8.1", art: "rain" },
  { title: "Circuit Breaker", genre: "Action · 2h 12m", rating: "8.5", art: "circuit" },
];

export function HomePage() {
  return <main>
    <section className="hero">
      <div className="hero-glow" />
      <div className="hero-copy"><span className="eyebrow">NOW PLAYING</span><h1>Stories worth<br /><em>showing up for.</em></h1><p>Find your next favourite film, choose your seats, and let the magic begin.</p><div className="hero-actions"><button className="button primary"><Play size={16} fill="currentColor" /> Book tickets</button><button className="button ghost">Explore movies <ArrowRight size={16} /></button></div></div>
      <div className="hero-poster"><div className="poster-sun" /><span>THE LAST<br />HORIZON</span><small>AN ARJUN MEHRA FILM</small></div>
    </section>
    <section className="quick-search"><div><MapPin size={18} /><span>Choose city</span><strong>Mumbai</strong></div><div><CalendarDays size={18} /><span>Date</span><strong>Today, 23 Aug</strong></div><button>Find movies <ArrowRight size={17} /></button></section>
    <section className="content-section"><div className="section-heading"><div><span className="eyebrow">CURATED FOR YOU</span><h2>Now showing</h2></div><a href="#movies">View all <ChevronRight size={17} /></a></div><div id="movies" className="movie-grid">{movies.map((movie) => <article className="movie-card" key={movie.title}><div className={`movie-art ${movie.art}`}><span className="age">UA 13+</span><button aria-label={`Play ${movie.title} trailer`}><Play size={15} fill="currentColor" /></button></div><div className="movie-info"><h3>{movie.title}</h3><p>{movie.genre}</p><span><Star size={14} fill="currentColor" /> {movie.rating}</span></div></article>)}</div></section>
  </main>;
}
