"""Synthesizes a 15s dark-tech soundtrack synced to scene.html cues. Writes soundtrack.wav."""
import numpy as np, wave, sys
SR = 48000; D = 15.0
t = np.arange(int(SR * D)) / SR
rng = np.random.default_rng(7)
out = np.zeros_like(t)

def env(t0, a, d):  # attack/decay envelope starting at t0
    x = t - t0
    e = np.where(x < 0, 0, np.where(x < a, x / a, np.exp(-(x - a) / d)))
    return e

# drone: detuned low saws filtered-ish via sines, swelling
drone_amp = np.clip(t / 3, 0, 1) * (1 - np.clip((t - 14.2) / 0.8, 0, 1))
for f, g in [(43.65, .35), (43.9, .3), (87.3, .12), (130.8, .05)]:
    out += g * drone_amp * np.sin(2 * np.pi * f * t + 0.3 * np.sin(2 * np.pi * 0.2 * t))

# pulse beat from 5.2s: kick every 0.5s
for k in np.arange(5.2, 14.0, 0.5):
    x = t - k; m = (x >= 0) & (x < 0.4)
    out[m] += 0.6 * np.sin(2 * np.pi * (50 + 120 * np.exp(-x[m] * 25)) * x[m]) * np.exp(-x[m] * 9)
# hats offbeat from 8.8s
noise = rng.standard_normal(len(t))
hp = noise - np.convolve(noise, np.ones(8) / 8, 'same')
for k in np.arange(9.05, 14.0, 0.5):
    out += 0.12 * hp * env(k, 0.002, 0.03)

# boot typing blips
for i, k in enumerate(np.arange(0.3, 2.4, 0.07)):
    f = 1800 + 600 * ((i * 7) % 5)
    out += 0.05 * np.sin(2 * np.pi * f * t) * env(k, 0.001, 0.015)
# riser into S completion
m = (t > 2.6) & (t < 5.2)
x = (t[m] - 2.6) / 2.6
out[m] += 0.12 * x ** 2 * np.sin(2 * np.pi * (200 * x + 800 * x ** 3) * (t[m] - 2.6)) + 0.05 * x ** 2 * noise[m]
# impacts
for k, g in [(5.2, 1.0), (13.3, 0.9)]:
    x = t - k; m = (x >= 0) & (x < 2.5)
    out[m] += g * 0.9 * np.sin(2 * np.pi * (35 + 80 * np.exp(-x[m] * 8)) * x[m]) * np.exp(-x[m] * 2.2)
    out += g * 0.25 * noise * env(k, 0.001, 0.25)
# glitch stutters
for k0 in [5.15, 8.8, 9.3, 12.1, 13.25]:
    for j in range(6):
        k = k0 + j * 0.04
        out += 0.08 * np.sign(np.sin(2 * np.pi * (300 + 200 * j) * t)) * env(k, 0.001, 0.02)
# tail fade
out *= 1 - np.clip((t - 14.5) / 0.5, 0, 1)
out /= np.max(np.abs(out)) * 1.12
st = np.stack([out, np.roll(out, 240) * 0.95], 1)  # slight stereo width
with wave.open(sys.argv[1] if len(sys.argv) > 1 else 'soundtrack.wav', 'wb') as w:
    w.setnchannels(2); w.setsampwidth(2); w.setframerate(SR)
    w.writeframes((st * 32767).astype(np.int16).tobytes())
