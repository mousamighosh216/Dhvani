package metadata

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"dhwani/recognition/fingerprint"
)

// Memory is a non-persistent Store for tests and running without a database.
type Memory struct {
	mu      sync.RWMutex
	songs   []Song
	idx     map[uint32][]Entry
	users   []User
	history []histRow
	cols    []colRow
	nextCol int
}

type histRow struct {
	id, user, song int
	conf           float64
	at             time.Time
}

type colRow struct {
	id, user int
	name     string
	songs    map[int]time.Time
}

func NewMemory() *Memory { return &Memory{idx: map[uint32][]Entry{}, nextCol: 1} }

func (m *Memory) AddSong(_ context.Context, _ int, s Song, fps []fingerprint.Fingerprint) (Song, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.songs {
		if x.Title == s.Title && x.Artist == s.Artist {
			return s, 0, ErrDuplicate
		}
	}
	s.ID = len(m.songs) + 1
	m.songs = append(m.songs, s)
	seen := map[[2]uint32]bool{}
	n := 0
	for _, f := range fps {
		k := [2]uint32{f.Hash, uint32(f.T)}
		if seen[k] {
			continue
		}
		seen[k] = true
		m.idx[f.Hash] = append(m.idx[f.Hash], Entry{SongID: s.ID, T: f.T})
		n++
	}
	return s, n, nil
}

func (m *Memory) Lookup(_ context.Context, hashes []uint32) (map[uint32][]Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[uint32][]Entry{}
	for _, h := range hashes {
		if e, ok := m.idx[h]; ok {
			out[h] = e
		}
	}
	return out, nil
}

func (m *Memory) GetSong(_ context.Context, id int) (Song, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.song(id)
}

func (m *Memory) song(id int) (Song, error) {
	if id < 1 || id > len(m.songs) {
		return Song{}, ErrNotFound
	}
	return m.songs[id-1], nil
}

func (m *Memory) SearchSongs(_ context.Context, q SongQuery) ([]Song, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	needle := strings.ToLower(q.Q)
	var all []Song
	for _, s := range m.songs {
		if needle != "" && !strings.Contains(strings.ToLower(s.Title+"\x00"+s.Artist+"\x00"+s.Genre), needle) {
			continue
		}
		if q.Genre != "" && !strings.EqualFold(s.Genre, q.Genre) {
			continue
		}
		all = append(all, s)
	}
	plays := map[int]int{}
	for _, h := range m.history {
		plays[h.song]++
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		switch q.Sort {
		case "artist":
			if x, y := strings.ToLower(a.Artist), strings.ToLower(b.Artist); x != y {
				return x < y
			}
			return a.Title < b.Title
		case "popular":
			if plays[a.ID] != plays[b.ID] {
				return plays[a.ID] > plays[b.ID]
			}
		}
		return a.ID > b.ID
	})
	if q.Offset > len(all) {
		q.Offset = len(all)
	}
	all = all[q.Offset:]
	if q.Limit < len(all) {
		all = all[:q.Limit]
	}
	if all == nil {
		all = []Song{}
	}
	return all, nil
}

func (m *Memory) Genres(_ context.Context) ([]GenreCount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c := map[string]int{}
	for _, s := range m.songs {
		if s.Genre != "" {
			c[s.Genre]++
		}
	}
	out := []GenreCount{}
	for g, n := range c {
		out = append(out, GenreCount{g, n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Genre < out[j].Genre
	})
	return out, nil
}

// ---- users ----

func (m *Memory) CreateUser(_ context.Context, name, email, hash string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return User{}, ErrEmailTaken
		}
	}
	u := User{ID: len(m.users) + 1, Name: name, Email: email, CreateDate: time.Now().UTC(), PasswordHash: hash}
	m.users = append(m.users, u)
	m.addCol(u.ID, FavouritesName)
	return u, nil
}

func (m *Memory) UserByEmail(_ context.Context, email string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (m *Memory) UserByID(_ context.Context, id int) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if id < 1 || id > len(m.users) {
		return User{}, ErrNotFound
	}
	return m.users[id-1], nil
}

func (m *Memory) UpdateUserName(_ context.Context, id int, name string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id < 1 || id > len(m.users) {
		return User{}, ErrNotFound
	}
	m.users[id-1].Name = name
	return m.users[id-1], nil
}

func (m *Memory) UserStats(_ context.Context, uid int) (UserStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var s UserStats
	now := time.Now().UTC()
	som := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	uniq := map[int]bool{}
	for _, h := range m.history {
		if h.user == uid {
			s.Identifications++
			uniq[h.song] = true
			if !h.at.Before(som) {
				s.ThisMonth++
			}
		}
	}
	s.UniqueSongs = len(uniq)
	saved := map[int]bool{}
	for _, c := range m.cols {
		if c.user == uid {
			s.Collections++
			for id := range c.songs {
				saved[id] = true
			}
		}
	}
	s.Saved = len(saved)
	return s, nil
}

// ---- history ----

func (m *Memory) AddHistory(_ context.Context, uid, songID int, conf float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = append(m.history, histRow{len(m.history) + 1, uid, songID, conf, time.Now().UTC()})
	return nil
}

func (m *Memory) ListHistory(_ context.Context, uid, limit int) ([]HistoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []HistoryItem{}
	for i := len(m.history) - 1; i >= 0 && len(out) < limit; i-- {
		h := m.history[i]
		if h.user != uid {
			continue
		}
		s, _ := m.song(h.song)
		out = append(out, HistoryItem{ID: h.id, Song: s, Confidence: h.conf, CreatedAt: h.at})
	}
	return out, nil
}

// ---- collections ----

func (m *Memory) addCol(uid int, name string) Collection {
	c := colRow{id: m.nextCol, user: uid, name: name, songs: map[int]time.Time{}}
	m.nextCol++
	m.cols = append(m.cols, c)
	return Collection{ID: c.id, Name: name}
}

func (m *Memory) Collections(_ context.Context, uid int) ([]Collection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Collection{}
	for _, c := range m.cols {
		if c.user == uid {
			out = append(out, Collection{c.id, c.name, len(c.songs)})
		}
	}
	return out, nil
}

func (m *Memory) CreateCollection(_ context.Context, uid int, name string) (Collection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.cols {
		if c.user == uid && c.name == name {
			return Collection{}, ErrNameTaken
		}
	}
	return m.addCol(uid, name), nil
}

func (m *Memory) CollectionSongs(_ context.Context, uid, colID int) ([]SavedSong, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	latest := map[int]SavedSong{}
	for _, c := range m.cols {
		if c.user != uid || (colID != 0 && c.id != colID) {
			continue
		}
		for id, at := range c.songs {
			s, _ := m.song(id)
			if cur, ok := latest[id]; !ok || at.After(cur.SavedAt) {
				latest[id] = SavedSong{Song: s, SavedAt: at}
			}
		}
	}
	out := []SavedSong{}
	for id, v := range latest {
		for i := len(m.history) - 1; i >= 0; i-- {
			if h := m.history[i]; h.user == uid && h.song == id {
				v.Confidence = h.conf
				break
			}
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].SavedAt.Equal(out[j].SavedAt) {
			return out[i].SavedAt.After(out[j].SavedAt)
		}
		return out[i].Song.ID > out[j].Song.ID
	})
	return out, nil
}

func (m *Memory) AddToCollection(_ context.Context, uid, colID, songID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.song(songID); err != nil {
		return err
	}
	for _, c := range m.cols {
		if c.id == colID && c.user == uid {
			if _, ok := c.songs[songID]; !ok {
				c.songs[songID] = time.Now().UTC()
			}
			return nil
		}
	}
	return ErrNotFound
}

func (m *Memory) RemoveFromCollection(_ context.Context, uid, colID, songID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.cols {
		if c.user == uid && (colID == 0 || c.id == colID) {
			delete(c.songs, songID)
		}
	}
	return nil
}
