package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "time"
    "strings"

    "github.com/jackc/pgx/v5/pgxpool"

    "moviehub/auth"
    "moviehub/handler"
    "moviehub/mw"
    "moviehub/store"
)

func main() {
    secret := os.Getenv("JWT_SECRET")
    if len(secret) < 32 {
        log.Fatal("JWT_SECRET ต้องยาวอย่างน้อย 32 ตัวอักษร")
    }
    pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()
    if err := pool.Ping(context.Background()); err != nil {
        log.Fatal("ต่อฐานข้อมูลไม่ติด ", err)
    }

    tickets := &auth.Issuer{Secret: []byte(secret), TTL: time.Hour}
    h := &handler.Handler{Store: &store.Store{Pool: pool}, Tickets: tickets}

    mux := http.NewServeMux()
    mux.HandleFunc("POST /api/auth/register", h.Register)
    mux.HandleFunc("POST /api/auth/login", h.Login)
    mux.HandleFunc("GET /api/movies", h.SearchMovies)
    mux.HandleFunc("GET /api/movies/{id}/reviews", h.ListReviews)

   // log.Println("MovieHub API ฟังอยู่ที่ :8080")
    //log.Fatal(http.ListenAndServe(":8080", mw.Logging(mux)))
    origins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
    log.Println("MovieHub API ฟังอยู่ที่ :8080 ต้อนรับ", origins)
    log.Fatal(http.ListenAndServe(":8080", mw.Logging(mw.CORS(origins, mux))))

    // MovieDetail คือหนังหนึ่งเรื่องพร้อมเรื่องย่อ ใช้กับหน้ารายละเอียด
type MovieDetail struct {
  Movie
  Detail *string `json:"detail"`
}

// GetMovie หาหนังจากเลข ไม่พบตอบ ErrNotFound
func (s *Store) GetMovie(ctx context.Context, id int64) (MovieDetail, error) {
  var m MovieDetail
  err := s.Pool.QueryRow(ctx,
    `SELECT m.id, m.title, m.title_th,
        (SELECT coalesce(g.name_th, g.name_en) FROM movie_genres mg
           JOIN genres g ON g.id = mg.genre_id
          WHERE mg.movie_id = m.id AND mg.position = 0),
        extract(year FROM m.release_date)::int,
        round(m.vote_average, 1)::float8,
        'https://image.tmdb.org/t/p/w500' || m.poster_path,
        coalesce(m.overview_th, m.overview)
     FROM movies m
     WHERE m.id = $1`, id).
    Scan(&m.ID, &m.Title, &m.TitleTh, &m.Genre, &m.Year, &m.Rating, &m.Poster, &m.Detail)
  if errors.Is(err, pgx.ErrNoRows) {
    return m, ErrNotFound
  }
  return m, err
}
}