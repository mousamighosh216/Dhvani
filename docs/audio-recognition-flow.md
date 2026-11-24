# Audio recognition communication flow

This sequence shows what happens after a user uploads an audio clip in Dhwani.
The catalog must already contain fingerprints for a song to be recognized; catalog
ingestion is a separate flow.

```mermaid
sequenceDiagram
    actor User
    participant UI as React chat (ChatWindow)
    participant Client as API client
    participant Router as Go HTTP router
    participant Handler as Recognition handler
    participant Decoder as Audio decoder
    participant Engine as Recognition engine
    participant DSP as DSP fingerprint pipeline
    participant Store as Fingerprint store (PostgreSQL or memory)
    participant Metadata as Song metadata store

    User->>UI: Select or drop audio file
    UI->>UI: Show audio message and "Listening" state
    UI->>Client: recognizeAudio(file, filename)
    Client->>Client: Build multipart form field "audio"
    Client->>Router: POST /api/v1/recognize
    Router->>Handler: HandleRecognize(request)
    Handler->>Handler: Limit request to 25 MB; parse multipart form

    alt Invalid form, missing audio, or oversized request
        Handler-->>Router: 400 JSON error
        Router-->>Client: Error response
        Client-->>UI: Reject with error message
        UI-->>User: Show error
    else Valid upload
        Handler->>Decoder: DecodeAudio(file)
        Note over Decoder: WAV uses native decoder; MP3 uses MP3 decoder;<br/>other formats use ffmpeg. Output is mono audio samples.
        alt Decode fails or clip is shorter than 1 second
            Decoder-->>Handler: Decode error (or short clip)
            Handler-->>Router: 400/503 JSON error
            Router-->>Client: Error response
            Client-->>UI: Reject with error message
            UI-->>User: Show error
        else Audio decoded and long enough
            Handler->>Engine: Recognize(context, clip)
            Engine->>DSP: Resample to 44.1 kHz; compute STFT
            DSP->>DSP: Extract spectral peaks and generate hashes
            DSP-->>Engine: Query fingerprints
            Engine->>Store: QueryHashes(query hashes)
            Store-->>Engine: Matching catalog hashes and song offsets
            Engine->>Engine: Vote on song/time-offset clusters
            Note over Engine: Accept winner only when it meets<br/>minimum votes and runner-up margin.

            alt Recognition or fingerprint-store error
                Engine-->>Handler: Error
                Handler-->>Router: 500 JSON error
                Router-->>Client: Error response
                Client-->>UI: Reject with error message
                UI-->>User: Show error
            else No convincing match
                Engine-->>Handler: No match
                Handler-->>Router: 200 success (matched=false)
                Router-->>Client: JSON response
                Client-->>UI: Resolve with matched=false
                UI-->>User: Show "couldn't find a match" guidance
            else Match found
                Engine-->>Handler: Song ID, confidence, votes, offset
                Handler->>Metadata: GetSongByID(song ID)
                alt Metadata lookup fails
                    Metadata-->>Handler: Error
                    Handler-->>Router: 500 JSON error
                    Router-->>Client: Error response
                    Client-->>UI: Reject with error message
                    UI-->>User: Show error
                else Metadata found
                    Metadata-->>Handler: Song, artist, album, links
                    Handler-->>Router: 200 success (matched=true + song details)
                    Router-->>Client: JSON response
                    Client-->>UI: Resolve with recognition result
                    UI-->>User: Show result card with song and match details
                end
            end
        end
    end

    Note over UI: Clear the busy state after success or error.
```
