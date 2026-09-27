package matcher

import "Dhvani/recognition/fingerprint"

type MemoryStore struct {
	index map[uint32][]DBMatch
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		index: make(map[uint32][]DBMatch),
	}
}

func (m *MemoryStore) IngestTrack(fps []fingerprint.Fingerprint) {
	for _, fp := range fps {
		m.index[fp.Hash] = append(m.index[fp.Hash], DBMatch{
			SongID:         fp.SongID,
			DatabaseOffset: fp.AbsoluteTime,
		})
	}
}

func (m *MemoryStore) QueryHashes(hashes []uint32) (map[uint32][]DBMatch, error) {
	result := make(map[uint32][]DBMatch)
	for _, h := range hashes {
		if matches, exists := m.index[h]; exists {
			result[h] = matches
		}
	}
	return result, nil
}
