* we started from audio accepting file which basically decodes audio file, that is, it converts audio to floating values normalized between -1 to 1.
* we are for now decoding 2 types of files
- wav: uncompressed PCM audio 

   - pcm - pulse code modulation (method used by computers to digitally represent analog audio signals).
   When sound travels through the air, it is a continuous analog wave of pressure changes. A microphone converts these pressure changes into continuous electrical voltages.

   how a microphone catches these signals?
   for a dynamic microphone, the diaphragm is attached to a small coil of wire sitting in a magnetic field.

    when it moves: 
    ```text
    air pressure changes
        ↓
    diaphragm moves
        ↓
    coil moves
        ↓
    magnetic flux through coil changes
        ↓
    Faraday's law
        ↓
    voltage is induced
    ```
    
* Because computers cannot store continuous infinite values, they must digitize the wave in two steps:
    - Sampling (Time Domain): The computer measures (samples) the amplitude of the electrical wave thousands of times per second. The number of measurements per second is the Sample Rate (measured in Hertz).
  
    - Common sample rate: 44,100 Hz (CD quality, meaning 44,100 individual measurements per second) or 22,050 Hz.
        - Quantization (Amplitude Domain): Each sampled measurement is converted into a discrete number (an integer). The precision of this number is determined by the Bit Depth.
            - 16-bit PCM: Amplitudes are stored as signed integers ranging from -32,768 to +32,767.
            - 24-bit PCM: Amplitudes range from -8,388,608 to +8,388,607.
                    
    Summary: PCM is simply a long array of integers representing the height (amplitude) of a sound wave sampled at regular time intervals.

    - what are mono samples?
    The term Mono (monophonic) refers to the number of audio channels being recorded or played back:
        - Stereo Audio (2 Channels): Has two separate channels—Left (L) and Right (R). In raw PCM memory, stereo data interleaves these channels:Memory:  [L_1, R_1, L_2, R_2, L_3, R_3,]
        - Mono Audio (1 Channel): Contains only a single stream of audio samples meant to be played equally through all speakers:Memory:  [S_1, S_2, S_3, S_4,]
        
- mp3: follows lossy compression 
    - wav contains pcm directly, so retrieval is direct
    - mp3 is decoded to pcm then forwarded to fft 
    
* Bit-Depth Normalization ($1/\text{maxVal}$):A 16-bit audio file records sample amplitudes as integers from $-32,768$ to $+32,767$. Dividing by $2^{15} = 32,768$ scales all amplitudes to $[-1.0, 1.0]$. This ensures volume variations do not break our frequency calculations later.

* Stereo to Mono Downmixing:If an audio file has left and right channels ($C_L, C_R$), we take the channel average for each frame:
$$S_{\text{mono}}[i] = \frac{C_L[i] + C_R[i]}{2}$$
This reduces the data volume by half while preserving all frequency components.

* **stft** 
a standard fft when applied on an entire audio file strips down the time domain while keeping frequency or note occured, if we want to identify which song is being played we need to know the frequency as well as the timestamp. 

This problem is solved by stft - short time fourier transform, which uses a sliding window approach. 

frame chunking and hop overlapping, Why:-
- Window Size ($N = 2048$): We take a slice of $2048$ audio samples. At $44.1\text{ kHz}$ sample rate, $2048$ samples corresponds to $\approx 46.4\text{ ms}$ of audio.
- Hop Size ($H = 512$): Instead of jumping forward by $2048$ samples for the next chunk, we only jump forward by $512$ samples ($\approx 11.6\text{ ms}$). This creates a $75\%$ overlap ($1536$ shared samples) between consecutive frames.
- Why overlap? Audio events like drum hits or fast guitar strums happen quickly. Overlapping ensures that transients occurring near the boundary of one frame aren't lost or distorted.

Applying the Hann Window Function:-
If you chop a raw audio wave abruptly into a $2048$-sample slice, the left and right edges will likely cut off mid-wave, creating sharp artificial step-discontinuities. In FFT mathematics, sharp sharp edges create fake high frequencies called Spectral Leakage.

To fix this, we multiply the $2048$ samples by a Hann Window curve before running FFT:
$$w[n] = 0.5 \times \left(1 - \cos\left(\frac{2\pi n}{N - 1}\right)\right)$$

The FFT Computation & Positive Bins:-
- Complex Numbers ($a + bi$): FFT outputs an array of $2048$ complex numbers.
- Symmetry: For real-valued audio signals, the second half of the FFT output ($1024$ to $2047$) is a mirrored complex conjugate of the first half ($0$ to $1023$). Thus, we discard the top half and keep only the first $N/2 = 1024$ frequency bins.
- Calculating Magnitude: For each bin $f$, we compute its magnitude (loudness):
$$\text{Magnitude} = \vert{}a + bi\vert{} = \sqrt{a^2 + b^2}$$

* we can kind off assume that its loosly like laplace, which converts complex equations into exponent based eqs. 