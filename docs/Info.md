
1. The Rule of Thumb Buffer Sizes

Depending on your application's tolerance for lag, choose a target duration:
• Ultra-Low Latency (5ms – 10ms): Used in professional live musical performance software or high-end local audio processing. Requires highly optimized networks and hardware.
• Real-Time Interactive (20ms – 40ms): The sweet spot. Humans perceive latency starting around 40-50ms. Keeping it at 20ms ensures smooth conversation even with slight network jitter. ~ 160 bytes 
• Standard Streaming (100ms – 1s+): Used for non-interactive playback (like internet radio or music players). High latency doesn't matter here, so large buffers ensure zero stuttering.

2. How Audio Formats Look in the First 12 Bytes

If you open an audio file in a hex editor (which shows raw bytes converted to text characters), this is exactly what is printed in those first few bytes:

    1. WAV Format
    A valid WAV file strictly follows the RIFF specification. Its first 12 bytes must follow this exact layout:
    • Bytes 0–3: RIFF (Signals it's a resource interchange file) [2]
    • Bytes 4–7: File size data (4 bytes of integers) [2]
    • Bytes 8–11: WAVE (Declares the format is specific audio) [2]
    If bytes 0–3 are RIFF and bytes 8–11 are WAVE, it is mathematically guaranteed to be a WAV file.

    2. MP3 Format
    MP3s often start with metadata tags called ID3v2 metadata.
    • Bytes 0–2: ID3
    • Byte 3: Version number (e.g., 0x03 or 0x04)
    Even if an MP3 doesn't have an ID3 tag, it starts with raw frame sync bytes: 0xFF 0xFB or 0xFF 0xF3. Your Go code only needs to look at the first 2 to 3 bytes to detect this.

    3. FLAC Format
    An open-source lossless FLAC file explicitly announces itself immediately.
    • Bytes 0–3: fLaC

3. Why Mono Samples are Relevant 

When building an audio fingerprinting system, stereo data introduces mathematical noise, triples processing overhead, and lowers matching accuracy.
    1. Acoustic Coherence: Audio fingerprinting relies on identifying absolute frequency peaks (like a snare hit or a vocal spike) on a 2D spectrogram. In a stereo track, an instrument might pan heavily to the left channel. If a user records that song in a room using their phone mic, the microphone captures an ambient blend of both speakers anyway. Fingerprinting in mono ensures your database matches what a real-world microphone actually hears.

    2. Eliminating Redundancy: Stereo audio contains two independent streams of data (Left and Right). Processing both channels means your Fast Fourier Transform (FFT) algorithm has to run twice as many math operations, and your PostgreSQL database has to store twice as many hashes. Converting to mono cuts your data processing and storage footprint in half with zero loss in recognition accuracy.

4. Example of a wav file

52 49 46 46  24 08 00 00  57 41 56 45  ...  64 61 74 61  10 00 00 00  00 00 00 00  00 40 00 C0
└─────────┬────────────┘  └────┬────┘       └─────────┬────────────┘  └───────────────────────┘
    "RIFF" + file size       "WAVE"               "data" chunk tag          Raw Audio Data bytes

    Because this file is 16-bit, every single sample takes up 2 bytes (16 bits / 8 = 2 bytes). Because it is a WAV file, it uses Little-Endian, meaning the bytes arrive backward.
    Here is how your loop maps those 8 raw bytes into your Go variables:

    Frame 0

    • Bytes 1 & 2 (00 00): This is Channel 1 (Left). Rearranged from little-endian, it is 0x0000, which equals integer 0.
    • Bytes 3 & 4 (00 00): This is Channel 2 (Right). Rearranged, it is 0x0000, which equals integer 0.
    • The Math: Sum = 0 + 0 = 0. Average sample = 0.
    • Stored in out[0]: 0.0 / 32768 = 0.0 (Complete silence)

    Frame 1

    • Bytes 5 & 6 (00 40): This is Channel 1 (Left). In little-endian, the 40 is the most significant byte, so it reads as 0x4000. In decimal, 0x4000 = integer 16384.
    • Bytes 7 & 8 (00 C0): This is Channel 2 (Right). In little-endian, this reads as 0xC000. Because 16-bit audio is signed, any hex number starting with C or F is a negative number. In two's complement, 0xC000 = integer -16384.
    • The Math: Sum = 16384 + (-16384) = 0. Average sample = 0.
    • Stored in out[1]: 0.0 / 32768 = 0.0 (The channels canceled each other out into mono silence)

5. Audio file → Audio samples → Spectrogram → Peaks → Fingerprints → Matching → Recognition 