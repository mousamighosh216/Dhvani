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
    - Sampling (Time Domain): The computer measures (samples) the amplitude of the electrical wave thousands of times per second. The number of measurements per second is the Sample Rate (measured in Hertz.
  
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
