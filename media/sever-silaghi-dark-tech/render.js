// Renders scene.html to PNG frames, then encodes MP4 with ffmpeg.
// Usage: node render.js <framesDir> [fps] [onlyTimes comma-separated]
const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');
(async () => {
  const out = process.argv[2];
  const fps = +(process.argv[3] || 30);
  const only = process.argv[4] ? process.argv[4].split(',').map(Number) : null;
  fs.mkdirSync(out, { recursive: true });
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1920, height: 1080 } });
  await page.addInitScript(() => { window.__RENDERING__ = true; });
  await page.goto('file://' + path.resolve(__dirname, 'scene.html'));
  const times = only || Array.from({ length: 15 * fps }, (_, i) => i / fps);
  for (let i = 0; i < times.length; i++) {
    const b64 = await page.evaluate(t => { render(t); return document.getElementById('c').toDataURL('image/png').split(',')[1]; }, times[i]);
    fs.writeFileSync(path.join(out, `f${String(i).padStart(4, '0')}.png`), Buffer.from(b64, 'base64'));
  }
  await browser.close();
})();
