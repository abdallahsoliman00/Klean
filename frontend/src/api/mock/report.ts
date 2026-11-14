/* The report the mock writes for "/report": several pages, with headings, tables (one long enough to
   cross a page), an inline SVG chart and a forced page break. It also carries a script, an event
   handler, a remote image and a link out, none of which may survive: a report is untrusted HTML. */

const ROUTES = [
  ['Harbour Loop', 412, 96.1, 2.1],
  ['North Pier Express', 388, 93.8, 3.4],
  ['Lighthouse Point', 301, 91.2, 4.8],
  ['Old Town Shuttle', 455, 97.4, 1.6],
  ['Outer Islands', 124, 84.7, 9.3],
] as const;

const MONTHS = [
  ['May', 95.2],
  ['Jun', 94.1],
  ['Jul', 90.3],
  ['Aug', 88.9],
  ['Sep', 93.6],
] as const;

const CAUSES = ['Weather hold', 'Late inbound vessel', 'Passenger loading', 'Engine check', 'Harbour traffic', 'Crew changeover'];

/** A sailing log long enough to run over a page. */
const sailings = Array.from({ length: 44 }, (_, i) => {
  const route = ROUTES[i % ROUTES.length][0];
  const day = String(1 + ((i * 3) % 28)).padStart(2, '0');
  const delay = 6 + ((i * 7) % 31);
  return `<tr><td>${day} Aug</td><td>${route}</td><td>${String(6 + (i % 13)).padStart(2, '0')}:${i % 2 ? '40' : '10'}</td><td class="n">${delay} min</td><td>${CAUSES[i % CAUSES.length]}</td></tr>`;
}).join('\n');

const bars = MONTHS.map(([month, share], i) => {
  const height = (share - 80) * 9;
  const x = 60 + i * 86;
  return `<rect x="${x}" y="${190 - height}" width="46" height="${height}" rx="3" fill="${share < 90 ? '#b4532a' : '#2f5d8a'}"/>
<text x="${x + 23}" y="${182 - height}" text-anchor="middle" font-size="12" fill="#1c2733">${share}%</text>
<text x="${x + 23}" y="210" text-anchor="middle" font-size="12" fill="#5b6672">${month}</text>`;
}).join('\n');

export const SAMPLE_REPORT = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Summer punctuality review</title>
<link rel="stylesheet" href="https://example.com/remote.css">
<style>
  @page { size: A4; margin: 14mm; }
  body { margin: 0; font: 11pt/1.5 Georgia, 'Times New Roman', serif; color: #1c2733; }
  h1, h2, h3, th, .meta, .kpis, figcaption { font-family: 'Helvetica Neue', Arial, sans-serif; }
  h1 { font-size: 24pt; line-height: 1.15; margin: 0 0 4pt; letter-spacing: -0.01em; }
  h2 { font-size: 14pt; margin: 22pt 0 6pt; padding-bottom: 4pt; border-bottom: 1.5pt solid #1c2733; }
  h3 { font-size: 11pt; margin: 14pt 0 4pt; }
  p { margin: 0 0 8pt; }
  .meta { color: #5b6672; font-size: 9.5pt; margin-bottom: 16pt; }
  .kpis { display: flex; gap: 10pt; margin: 14pt 0 6pt; }
  .kpis div { flex: 1; border: 1pt solid #d5dbe2; border-radius: 4pt; padding: 8pt 10pt; }
  .kpis b { display: block; font-size: 17pt; }
  .kpis span { font-size: 8.5pt; color: #5b6672; }
  table { width: 100%; border-collapse: collapse; font-size: 9.5pt; margin: 6pt 0 10pt; }
  th { text-align: left; font-size: 8.5pt; color: #5b6672; border-bottom: 1pt solid #1c2733; padding: 5pt 6pt; }
  td { padding: 5pt 6pt; border-bottom: 0.5pt solid #d5dbe2; }
  tr { break-inside: avoid; }
  .n, th.n { text-align: right; font-variant-numeric: tabular-nums; }
  figure { margin: 10pt 0; }
  figcaption { font-size: 8.5pt; color: #5b6672; margin-top: 4pt; }
  .new-page { break-before: page; }
  ol { margin: 0 0 8pt; padding-left: 16pt; }
  li { margin-bottom: 5pt; }
</style>
</head>
<body>
<script>document.body.innerHTML = 'A script ran in the report.';</script>
<h1>Summer punctuality review</h1>
<div class="meta">Harbour Ferry Service, May to September 2026. Prepared for the operations board.</div>

<p>Across the summer timetable the fleet ran 1,680 scheduled sailings and 93.4% of them left within five minutes of the published time. That is 1.2 points below last summer. Almost all of the gap opened in July and August, and most of it on a single route.</p>

<div class="kpis">
  <div><b>93.4%</b><span>sailings on time</span></div>
  <div><b>1,680</b><span>scheduled sailings</span></div>
  <div><b>3.6 min</b><span>average delay</span></div>
  <div><b>11</b><span>cancellations</span></div>
</div>

<h2>Punctuality by route</h2>
<p>Four of the five routes met the 90% target. Outer Islands did not: it is the longest crossing, the most exposed to weather, and the only one served by a single vessel, so one late arrival carries through the rest of the day.</p>
<table>
  <thead><tr><th>Route</th><th class="n">Sailings</th><th class="n">On time</th><th class="n">Average delay</th></tr></thead>
  <tbody>
${ROUTES.map(([name, sailingsRun, share, delay]) => `    <tr><td>${name}</td><td class="n">${sailingsRun}</td><td class="n">${share}%</td><td class="n">${delay} min</td></tr>`).join('\n')}
  </tbody>
</table>

<h2>Month by month</h2>
<figure>
  <svg viewBox="0 0 520 220" width="100%" role="img" aria-label="On-time sailings by month">
    <line x1="40" y1="190" x2="500" y2="190" stroke="#1c2733" stroke-width="1"/>
    <line x1="40" y1="100" x2="500" y2="100" stroke="#b9c2cc" stroke-width="1" stroke-dasharray="4 4"/>
    <text x="500" y="94" text-anchor="end" font-size="11" fill="#5b6672">90% target</text>
${bars}
  </svg>
  <figcaption>Share of sailings that left within five minutes of schedule. July and August fell below the target.</figcaption>
</figure>
<p>The dip follows the school holidays almost exactly. Loading times on the two busiest routes grew by about ninety seconds per sailing, which the turnaround allowance could not absorb.</p>
<img src="https://example.com/tracking-pixel.png" onerror="document.body.innerHTML = 'An event handler ran in the report.'" alt="">

<h2>Sailing log: August delays over five minutes</h2>
<p>Every August sailing that left more than five minutes late, with the cause the duty officer recorded. The full log is kept by <a href="https://example.com/ops">the operations desk</a>.</p>
<table>
  <thead><tr><th>Date</th><th>Route</th><th>Departure</th><th class="n">Delay</th><th>Recorded cause</th></tr></thead>
  <tbody>
${sailings}
  </tbody>
</table>

<h2 class="new-page">What caused the delays</h2>
<p>Weather holds account for a third of the delayed minutes but only a tenth of the delayed sailings: they are rare and long. Passenger loading is the reverse, many short delays that add up over a day. The two need different fixes.</p>
<h3>Loading</h3>
<p>Boarding through a single gangway is the limit on the Harbour Loop and Old Town Shuttle. A second gangway was trialled for two weeks in September and cut the average loading time from 6.5 to 4.2 minutes.</p>
<h3>Knock-on delays</h3>
<p>On Outer Islands, 71% of late departures followed a late arrival of the same vessel. A standby vessel on peak days would break that chain.</p>

<h2>Recommendations</h2>
<ol>
  <li><b>Fit second gangways</b> on the two busiest routes before next May. The trial paid for itself in recovered minutes.</li>
  <li><b>Add ten minutes of turnaround</b> to the Outer Islands timetable in July and August, when the weather is least settled.</li>
  <li><b>Charter a standby vessel</b> for the eight peak weekends, to stop one late crossing delaying the rest of the day.</li>
  <li><b>Record causes as they happen</b>, on the bridge tablet, so that next year's review does not depend on memory.</li>
</ol>
<p>Taken together these should bring the summer figure back above 95%.</p>
</body>
</html>`;
