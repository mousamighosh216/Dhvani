package metadata

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"dhwani/recognition/fingerprint"
	"dhwani/recognition/matcher"
)

// MemoryStore implements Service and matcher.FingerprintStore in RAM.
// It backs the unit tests and the `nodeps` build; production uses PostgresStore.
type MemoryStore struct {
	mu     sync.RWMutex
	nextID int
	songs  map[int]SongMetadata
	index  map[uint32][]matcher.DBMatch
}

// NewMemoryStore returns an empty store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1, songs: map[int]SongMetadata{}, index: map[uint32][]matcher.DBMatch{}}
}

func (m *MemoryStore) IngestSong(_ context.Context, song *SongMetadata, fps []fingerprint.Fingerprint) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextID
	m.nextID++
	s := *song
	s.ID = id
	s.CreatedAt = time.Now().UTC()
	s.Artist.ID = id // synthetic ids, good enough for tests
	if s.Album != nil {
		a := *s.Album
		a.ID = id
		s.Album = &a
	}
	if s.ExternalLinks == nil {
		s.ExternalLinks = map[string]string{}
	}
	m.songs[id] = s
	seen := map[[2]uint32]bool{}
	for _, fp := range fps {
		k := [2]uint32{fp.Hash, fp.SongOffset}
		if seen[k] {
			continue
		}
		seen[k] = true
		m.index[fp.Hash] = append(m.index[fp.Hash], matcher.DBMatch{Hash: fp.Hash, SongID: id, SongOffset: fp.SongOffset})
	}
	return id, nil
}

func (m *MemoryStore) GetSongByID(_ context.Context, id int) (*SongMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.songs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &s, nil
}

func (m *MemoryStore) ListSongs(_ context.Context, limit, offset int, sortBy string) ([]SongMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := make([]SongMetadata, 0, len(m.songs))
	for _, s := range m.songs {
		all = append(all, s)
	}
	if sortBy == "artist" {
		sort.Slice(all, func(i, j int) bool {
			artistI, artistJ := strings.ToLower(all[i].Artist.Name), strings.ToLower(all[j].Artist.Name)
			if artistI != artistJ {
				return artistI < artistJ
			}
			titleI, titleJ := strings.ToLower(all[i].Title), strings.ToLower(all[j].Title)
			if titleI != titleJ {
				return titleI < titleJ
			}
			return all[i].ID < all[j].ID
		})
	} else {
		sort.Slice(all, func(i, j int) bool { return all[i].ID > all[j].ID })
	}
	if offset > len(all) {
		offset = len(all)
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}

func (m *MemoryStore) QueryHashes(_ context.Context, hashes []uint32) ([]matcher.DBMatch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []matcher.DBMatch
	for _, h := range hashes {
		out = append(out, m.index[h]...)
	}
	return out, nil
}

// SaveFingerprints appends fingerprints for existing song IDs.
func (m *MemoryStore) SaveFingerprints(_ context.Context, fps []fingerprint.Fingerprint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, fp := range fps {
		m.index[fp.Hash] = append(m.index[fp.Hash], matcher.DBMatch{Hash: fp.Hash, SongID: fp.SongID, SongOffset: fp.SongOffset})
	}
	return nil
}
