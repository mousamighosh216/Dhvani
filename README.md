# Dhvani 🎵 — Music Recognition & Discovery Platform

**Dhvani** is a high-performance music recognition and discovery platform powered by classical Digital Signal Processing (DSP) audio fingerprinting and time-offset voting. Built with a modular **Go** backend monolith and a modern **React** chat interface, Dhumani transforms raw audio recordings into identified songs and rich music recommendations.

---

## 🚀 Key Features

* **Classical Audio Fingerprinting (Phase 1)**: Robust song identification using FFT, STFT, spectral peak detection, anchor-target pairing, and temporal offset voting.
* **Noise-Resistant Identification**: High accuracy even in the presence of background noise, volume fluctuations, and microphone coloration.
* **Conversational Chat Interface**: ChatGPT-style React UI with Web Audio API recording, drag-and-drop file uploading, and real-time waveforms.
* **Content-Based Recommendations (Phase 2)**: Acoustic feature extraction (BPM, MFCCs, Chroma) to suggest musically similar tracks.
* **Modular Monolith Architecture**: Decoupled Go packages ensuring parallel development between algorithm (Developer A) and product (Developer B) layers.

---

## 📂 Project Directory Structure

```text
dhvani/
├── README.md
├── docker-compose.yml
├── docs/
│   ├── system-design.md
│   └── api-spec.md
├── database/
│   └── migrations/
│       └── 001_initial_schema.sql
├── frontend/                         
│   ├── public/
│   └── src/
│       ├── components/               # Chat UI, Audio Recorder, Song Cards
│       ├── hooks/                    # Web Audio API hooks
│       ├── services/                 # API client
│       └── App.jsx
└── backend/                          # Go Monolith
    ├── go.mod
    ├── go.sum
    ├── main.go
    ├── api/                          # Developer B (HTTP Router & Handlers)
    │   ├── router.go
    │   └── recognition_handler.go
    ├── metadata/                     # Developer B (DB Persistence & Models)
    │   ├── postgres.go
    │   └── song_service.go
    └── recognition/                 
        ├── audio/                    # WAV/PCM decoding & normalization
        │   └── decoder.go
        ├── fft/                      # STFT & Spectrogram computation
        ├── peaks/                    # 2D Spectral peak extraction
        ├── fingerprint/              # Anchor-target peak pairing & hashing
        └── matcher/                  # Index retrieval & time-offset voting
```
---

## 📐 System Architecture & Diagrams
### 1. System Architecture
Dhumani separates client interaction, API orchestration, core signal processing, and database indexing into clean layers:

```text
+-------------------------------------------------------------------------------+
|                             CLIENT / FRONTEND                                 |
|  React Web App (Chat UI, Mic Recorder, Audio Player, Upload UI)               |
+-------------------------------------------------------------------------------+
                                      |
                           HTTP / REST API (JSON / FormData)
                                      v
+-------------------------------------------------------------------------------+
|                            GO BACKEND MONOLITH                                |
|                                                                               |
|  +-------------------------------------------------------------------------+  |
|  |                         API Gateway / HTTP Router                       |  |
|  +-------------------------------------------------------------------------+  |
|                          |                            |                       |
|                          v                            v                       |
|  +-------------------------------+    +------------------------------------+  |
|  |    Application & Meta Module   |    |         Recognition Core           |  |
|  |         (Developer B)         |    |           (Developer A)            |  |
|  |  - Song / Metadata Services   |    |  - Audio Decoder / Preprocessor    |  |
|  |  - History & Saved Songs      |    |  - DSP Pipeline (FFT/STFT)          |  |
|  |  - Recommendation Engine (P2) |    |  - Peak Detector & Fingerprinter   |  |
|  |  - External Link Resolver     |    |  - Time-Offset Matcher             |  |
|  +-------------------------------+    +------------------------------------+  |
|                  |                                      |                     |
+------------------|--------------------------------------|---------------------+
                   |                                      |
                   v                                      v
+-------------------------------------------------------------------------------+
|                               PERSISTENCE LAYER                               |
|                                                                               |
|  PostgreSQL Database                                                          |
|   ├── Metadata Store: songs, artists, albums, history, user_saved              |
|   └── Recognition Index: fingerprints (hash -> song_id, offset)              |
|                                                                               |
|  Local Filesystem / Object Storage                                            |
|   └── Audio Files (.mp3 / .wav source files)                                  |
+-------------------------------------------------------------------------------+
```
### 2. Class / Component Diagram (Go Interfaces)
The interface contract decouples the backend development:

```text
+------------------------------------------------------------------------------------+
|                                package: recognition                                |
|                                   (Developer A)                                    |
+------------------------------------------------------------------------------------+
|  <<interface>>                                                                     |
|  Engine                                                                            |
|  + Recognize(pcmData []byte) (*MatchResult, error)                                 |
|  + FingerprintSong(songID int, pcmData []byte) ([]Fingerprint, error)             |
+------------------------------------------------------------------------------------+
                                         ^
                                         | implements
+------------------------------------------------------------------------------------+
|  DefaultEngine                                                                     |
+------------------------------------------------------------------------------------+
|  - preprocessor : AudioPreprocessor                                                |
|  - stft         : STFTAnalyzer                                                     |
|  - peakDetector : PeakDetector                                                     |
|  - fingerprinter: FingerprintGenerator                                             |
|  - store        : FingerprintStore                                                 |
+------------------------------------------------------------------------------------+
|  + Recognize(pcmData []byte) (*MatchResult, error)                                 |
|  + FingerprintSong(songID int, pcmData []byte) ([]Fingerprint, error)             |
+------------------------------------------------------------------------------------+
       |                  |                  |                  |
       v                  v                  v                  v
+--------------+   +--------------+   +--------------+   +-------------------+
| AudioPreproc |   | STFTAnalyzer |   | PeakDetector |   | FingerprintGen    |
+--------------+   +--------------+   +--------------+   +-------------------+
| + ToMono()   |   | + Compute()  |   | + FindPeaks()|   | + PairPeaks()     |
| + Resample() |   |   ->Spectro  |   |   ->Landmarks|   | + HashTuples()    |
+--------------+   +--------------+   +--------------+   +-------------------+

                                         | uses
                                         v
                      +-------------------------------------+
                      |       <<interface>>                 |
                      |       FingerprintStore              |
                      +-------------------------------------+
                      | + QueryHashes(hashes []uint32)      |
                      |   -> []DBMatch                      |
                      | + SaveFingerprints(fps []Fingerprint|
                      +-------------------------------------+
                                         ^
                                         | implements
                                         |
+------------------------------------------------------------------------------------+
|                                 package: metadata                                  |
|                                   (Developer B)                                    |
+------------------------------------------------------------------------------------+
|  PostgresStore                                                                     |
+------------------------------------------------------------------------------------+
|  - db : *sql.DB                                                                    |
+------------------------------------------------------------------------------------+
|  + QueryHashes(hashes []uint32) ([]DBMatch, error)                                 |
|  + GetSongMetadata(songID int) (*SongMetadata, error)                              |
+------------------------------------------------------------------------------------+
```

### 3. Entity-Relationship (ER) Diagram
PostgreSQL schema supporting both rich application metadata and high-speed recognition lookup indexing:

```text
+-------------------+           +-------------------+
|      ARTISTS      |           |      ALBUMS       |
+-------------------+           +-------------------+
| id (PK)           |<----1---N-| id (PK)           |
| name              |           | artist_id (FK)    |
| bio               |           | title             |
+-------------------+           | release_date      |
          ^                     | cover_art_url     |
          |                     +-------------------+
          | 1                             | 1
          |                               |
          | N                             | N
+---------------------------------------------------+
|                       SONGS                       |
+---------------------------------------------------+
| id (PK)                                           |
| artist_id (FK)                                    |
| album_id (FK)                                     |
| title                                             |
| duration_seconds                                  |
| source_file_path                                  |
| external_links (JSONB: Spotify, YouTube, etc.)    |
| created_at                                        |
+---------------------------------------------------+
      | 1                                 | 1
      |                                   |
      | N                                 | N
+-------------------+           +-------------------+
|    FINGERPRINTS   |           |   SONG_FEATURES   |  <-- Phase 2 Recommendations
+-------------------+           +-------------------+
| hash (PK, Index)  |           | song_id (PK, FK)  |
| song_id (PK, FK)  |           | tempo             |
| song_offset (PK)  |           | energy            |
+-------------------+           | spectral_centroid |
                                | mfcc_vector       |
                                +-------------------+
```
## 📡 REST API Specifications
### POST /api/v1/recognize
Submits an audio sample (file upload or microphone blob) for identification.
* Content-Type: multipart/form-dataForm 
* Field: audio (file, required)
Sample Response (200 OK - Match Found):

```Bash
{
  "status": "success",
  "data": {
    "matched": true,
    "confidence": 0.94,
    "match_offset_seconds": 42.5,
    "song": {
      "id": 101,
      "title": "Midnight City",
      "duration_seconds": 243,
      "artist": { "id": 12, "name": "M83" },
      "album": {
        "id": 45,
        "title": "Hurry Up, We're Dreaming",
        "cover_art_url": "[https://cdn.example.com/covers/45.jpg](https://cdn.example.com/covers/45.jpg)",
        "release_date": "2011-10-18"
      },
      "external_links": {
        "spotify": "[https://open.spotify.com/track/6RUK21P2P03xD3399TF3Oj](https://open.spotify.com/track/6RUK21P2P03xD3399TF3Oj)",
        "youtube": "[https://www.youtube.com/watch?v=dX3k_QDnzHE](https://www.youtube.com/watch?v=dX3k_QDnzHE)"
      }
    }
  }
}
```
### POST /api/v1/songs
Ingests a new audio track into the catalog and extracts audio fingerprints.  

### GET /api/v1/songs/:id/recommendations
Returns musically similar tracks based on acoustic feature vector similarity.  

## 🔬 Mathematical Implementation Overview
The DSP pipeline follows a 5-step classical recognition workflow:  
```math
Audio Decoding & Normalization: Stereo channels are averaged to mono $S_{\text{mono}} = \frac{C_L + C_R}{2}$ and scaled into floating-point range $[-1.0, 1.0]$.
Short-Time Fourier Transform (STFT): Audio is split into overlapping windows ($N=2048$, $50\%$ overlap). FFT converts each time frame into frequency magnitude spectrums to build a 2D Spectrogram[cite: 1].
Spectral Peak Detection: Strong local maxima landmarks are extracted across time-frequency bins[cite: 1].
Anchor-Target Pairing & Hashing: Landmark pairs within a constrained bounding box are packed into compact 32-bit hashes:$$\text{Hash} = (f_1 \ll 16) \mid (f_2 \ll 8) \mid \Delta t$$
Time-Offset Voting: Matches are evaluated by temporal offset alignment:$$\text{Offset} = t_{\text{database}} - t_{\text{query}}$$
Candidate tracks are ranked by the density of their largest offset cluster[cite: 1].
```

## 🛠️ Getting Started

Prerequisites
* Go (v1.22 or higher)
* Node.js (v18 or higher)
* PostgreSQL (v15 or higher)

Installation & Run
* Clone Repository:
```Bash
 git clone [https://github.com/mousamighosh216/dhvani.git](https://github.com/mousamighosh216/dhvani.git)
cd dhvani
```
Database Setup:
```Bash
psql -U postgres -d dhumani -f database/migrations/001_initial_schema.sql
```
Backend Setup:
```Bash 
cd backend
go mod download
go run main.go
```
Frontend Setup:
```Bash
cd ../frontend
npm install
npm run dev
```
---