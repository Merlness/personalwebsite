// Home page star chart: builds the sky, the constellations that link to each
// part of the site, the eyepiece panel, and the moon phase of the day.
(function () {
  var NS = 'http://www.w3.org/2000/svg';
  function el(name, attrs, parent) {
    var node = document.createElementNS(NS, name);
    for (var k in attrs) node.setAttribute(k, attrs[k]);
    if (parent) parent.appendChild(node);
    return node;
  }

  // Cover: open once per visit, never for a reader who skips or comes back.
  var body = document.documentElement;
  try { if (sessionStorage.getItem('firstLight')) body.classList.add('sc-seen'); } catch (e) {}
  var openBtn = document.getElementById('sc-open');
  var chartStage = document.getElementById('chart');
  function open() {
    if (body.classList.contains('sc-open')) return;
    body.classList.add('sc-open');
    try { sessionStorage.setItem('firstLight', '1'); } catch (e) {}
    chartStage.focus({ preventScroll: true });
  }
  if (openBtn) openBtn.addEventListener('click', open);

  // The name in the header goes back to the beginning: the cover closes again.
  function showCover() {
    try { sessionStorage.removeItem('firstLight'); } catch (e) {}
    body.classList.remove('sc-seen', 'sc-open');
    window.scrollTo(0, 0);
    if (openBtn) openBtn.focus();
  }
  var brand = document.getElementById('brand');
  if (brand) brand.addEventListener('click', function (e) { e.preventDefault(); showCover(); });

  // Background stars: a fixed seed, so the sky is the same on every visit.
  var seed = 20200607;
  function rnd() { seed = (seed * 1664525 + 1013904223) % 4294967296; return seed / 4294967296; }
  var field = document.getElementById('field');
  for (var i = 0; i < 420; i++) {
    var a = rnd() * Math.PI * 2, r = Math.sqrt(rnd()) * 620, m = rnd();
    el('circle', { class: 'star', cx: (500 + Math.cos(a) * r).toFixed(1), cy: (500 + Math.sin(a) * r).toFixed(1),
      r: (m > 0.97 ? 2.4 : m > 0.85 ? 1.5 : 0.8).toFixed(1), opacity: (0.35 + m * 0.6).toFixed(2) }, field);
  }
  var spokes = document.getElementById('spokes');
  for (var h = 0; h < 12; h++) {
    var t = h * Math.PI / 6;
    el('line', { class: 'grid', x1: 500, y1: 500, x2: 500 + Math.cos(t) * 440, y2: 500 + Math.sin(t) * 440 }, spokes);
  }

  // Each constellation is one part of the site.
  // Each constellation is one part of the site, and its stars draw that part:
  // a camera, a mountain range, an open notebook, code brackets, a telescope.
  // "lines" are separate strokes, each a list of star numbers.
  var parts = [
    { id: 'photo', dive: [345, 336], title: 'Photography', sub: 'la cámara', href: '/portfolio', go: 'Open the plates →',
      text: 'Landscapes, wildlife, and the night sky.',
      img: '/assets/portfolio/RoadTrip2020/DSC02738_w1600.jpg',
      data: [['Object', 'Comet NEOWISE'], ['Site', 'Sequoia, California'], ['Date', '2020 Jul 15']],
      stars: [[255, 285], [300, 285], [318, 256], [372, 256], [390, 285], [435, 285], [435, 385], [255, 385],
              [345, 303], [374, 320], [374, 352], [345, 369], [316, 352], [316, 320], [345, 336]],
      lines: [[0, 1, 2, 3, 4, 5, 6, 7, 0], [8, 9, 10, 11, 12, 13, 8]], lead: 14, label: [255, 215] },
    { id: 'journeys', dive: [745, 250], title: 'Journeys', sub: 'el camino', href: '/journeys', go: 'Follow a route →',
      text: 'Trips you can follow on a map, stop by stop.',
      img: '/assets/portfolio/RoadTrip2020/DSC00306_w1600.jpg',
      data: [['Object', 'Maroon Bells'], ['Trip', 'Road Trip 2020'], ['Odometer', '9,000 miles']],
      stars: [[585, 400], [650, 285], [690, 345], [745, 250], [825, 400], [632, 318], [650, 332], [668, 314], [722, 290], [745, 306], [768, 292]],
      lines: [[0, 1, 2, 3, 4], [5, 6, 7], [8, 9, 10]], lead: 3, label: [640, 205] },
    { id: 'journal', dive: [300, 648], title: 'Journal', sub: 'el cuaderno', href: '/blog', go: 'Read the entries →',
      text: 'Stories from the river, the road, and the wild.',
      img: '/assets/portfolio/Alaska/DSC05913_w1600.jpg',
      data: [['Object', 'Bristol Bay'], ['Season', 'Alaska, 2018'], ['Entries', '3']],
      stars: [[300, 590], [210, 565], [210, 680], [300, 705], [390, 680], [390, 565], [232, 610], [278, 622], [232, 645], [278, 657], [322, 622], [368, 610]],
      lines: [[0, 1, 2, 3, 0], [0, 5, 4, 3], [6, 7], [8, 9], [10, 11]], lead: 0, label: [205, 750] },
    { id: 'tech', dive: [685, 695], title: 'Tech', sub: 'la máquina', href: 'https://bennusystems.com', go: 'See the work →',
      text: 'Merl Martin is the founder and CEO of Bennu Systems, an AI advisory firm in Scottsdale, Arizona. A former physics teacher turned software craftsman, he helps companies use AI with real discipline and teaches their people to use it themselves. He also built Enki Follow Up, a follow-up app for networkers, now in development.',
      img: '/assets/aboutme/headshot_w1600.jpg',
      data: [['Founder', 'Bennu Systems'], ['Field', 'AI advisory'], ['Base', 'Scottsdale, Arizona']],
      stars: [[630, 640], [585, 695], [630, 750], [700, 625], [670, 765], [740, 640], [785, 695], [740, 750]],
      lines: [[0, 1, 2], [3, 4], [5, 6, 7]], lead: 6, label: [640, 815] },
    { id: 'about', dive: [548, 428], title: 'About', sub: 'el observador', href: '/about', go: 'Meet the observer →',
      text: 'Physics teacher turned software craftsman.',
      img: '/assets/aboutme/profile_w600.jpg',
      data: [['Observer', 'Merl Martin'], ['Station', 'Scottsdale, Arizona'], ['Lat, Lon', '33.49° N, 111.93° W']],
      stars: [[452, 512], [548, 428], [560, 442], [464, 526], [500, 486], [470, 560], [500, 568], [530, 560]],
      lines: [[0, 1, 2, 3, 0], [4, 5], [4, 6], [4, 7]], lead: 1, label: [468, 615] }
  ];

  // Dive in: fly the chart into the clicked shape, then open its page.
  var chart = document.querySelector('.sc .chart'), wrap = document.querySelector('.sc');
  var calm = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  function dive(e, part) {
    if (calm || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button) return; // plain link
    e.preventDefault();
    if (wrap.classList.contains('acting') || wrap.classList.contains('diving')) return;
    // First a short action in the shape itself, then the dive.
    var link = e.currentTarget, act = el('g', { class: 'act act-' + part.id }, link);
    (acts[part.id] || function () {})(act, part);
    wrap.classList.add('acting');
    setTimeout(function () {
      chart.style.transformOrigin = (part.dive[0] / 10) + '% ' + (part.dive[1] / 10) + '%';
      wrap.classList.add('diving');
      body.classList.add('sc-diving');
    }, 520);
    setTimeout(function () { window.location.href = part.href; }, 1400);
  }

  function pts(part, list) { return list.map(function (i) { return part.stars[i].join(','); }).join(' '); }
  var acts = {
    // The shutter: the lens closes for an instant and the flash fires.
    photo: function (g, part) {
      el('polygon', { class: 'shutter', points: pts(part, [8, 9, 10, 11, 12, 13]) }, g);
      el('circle', { class: 'flash', cx: part.dive[0], cy: part.dive[1], r: 30 }, g);
    },
    // A climb: a marigold line runs up the ridge and a flag goes up on top.
    journeys: function (g, part) {
      el('polyline', { class: 'climb', points: pts(part, [0, 1, 2, 3]), pathLength: 100 }, g);
      var top = part.stars[3];
      el('path', { class: 'flag', d: 'M' + top[0] + ' ' + top[1] + 'v-34l22 8-22 9' }, g);
    },
    // A page turns over the spine.
    journal: function (g, part) {
      el('polygon', { class: 'page', points: pts(part, [0, 5, 4, 3]) }, g);
    },
    // Someone types: three marks appear between the brackets.
    tech: function (g) {
      [655, 685, 715].forEach(function (x, i) {
        el('rect', { class: 'key k' + i, x: x - 7, y: 688, width: 14, height: 14, rx: 2 }, g);
      });
    },
    // The telescope finds its star.
    about: function (g, part) {
      var a = part.stars[1];
      el('line', { class: 'beam', x1: a[0], y1: a[1], x2: a[0] + 78, y2: a[1] - 68, pathLength: 100 }, g);
      el('path', { class: 'found', d: 'M' + (a[0] + 78) + ' ' + (a[1] - 84) + 'v32M' + (a[0] + 62) + ' ' + (a[1] - 68) + 'h32' }, g);
    }
  };
  // Coming back with the Back button: put the chart where it was.
  window.addEventListener('pageshow', function () {
    wrap.classList.remove('diving', 'acting');
    body.classList.remove('sc-diving');
    Array.prototype.forEach.call(document.querySelectorAll('.sc .act'), function (n) { n.remove(); });
  });

  var layer = document.getElementById('constellations');
  var plate = { img: document.getElementById('p-img'), title: document.getElementById('p-title'), text: document.getElementById('p-text'),
    data: document.getElementById('p-data'), go: document.getElementById('p-go') };
  var links = [];

  function show(part, link) {
    links.forEach(function (l) { l.classList.toggle('on', l === link); });
    plate.img.src = part.img; plate.img.alt = part.data[0][1];
    plate.title.textContent = part.title; plate.text.textContent = part.text;
    plate.go.textContent = part.go; plate.go.href = part.href;
    plate.data.innerHTML = '';
    part.data.forEach(function (row) {
      var dt = document.createElement('dt'); dt.textContent = row[0];
      var dd = document.createElement('dd'); dd.textContent = row[1];
      plate.data.appendChild(dt); plate.data.appendChild(dd);
    });
  }

  parts.forEach(function (part, index) {
    var link = el('a', { class: 'con', href: part.href, 'aria-label': part.title }, layer);
    var xs = part.stars.map(function (s) { return s[0]; }), ys = part.stars.map(function (s) { return s[1]; });
    el('rect', { class: 'hit', x: Math.min.apply(null, xs) - 34, y: Math.min.apply(null, ys) - 34,
      width: Math.max.apply(null, xs) - Math.min.apply(null, xs) + 68, height: Math.max.apply(null, ys) - Math.min.apply(null, ys) + 68, rx: 10 }, link);
    part.lines.forEach(function (stroke) {
      for (var n = 1; n < stroke.length; n++) {
        var from = part.stars[stroke[n - 1]], to = part.stars[stroke[n]];
        el('line', { x1: from[0], y1: from[1], x2: to[0], y2: to[1] }, link);
      }
    });
    part.stars.forEach(function (s, i) {
      el('circle', { cx: s[0], cy: s[1], r: i === part.lead ? 6 : 3, class: i === part.lead ? 'lead' : '' }, link);
    });
    el('text', { x: part.label[0], y: part.label[1] }, link).textContent = part.title;
    el('text', { class: 'sub', x: part.label[0], y: part.label[1] + 22 }, link).textContent = part.sub;
    links.push(link);
    link.addEventListener('click', function (e) { dive(e, part); });
    link.addEventListener('mouseenter', function () { show(part, link); });
    link.addEventListener('focus', function () { show(part, link); });
    if (index === 0) show(part, link);
  });

  // Supernovae: about half as often as the meteors, one star flares and fades.
  if (!calm) {
    var disc = document.querySelector('#spokes').parentNode;
    var nova = function () {
      var a = Math.random() * Math.PI * 2, r = Math.sqrt(Math.random()) * 400;
      var g = el('g', { class: 'nova', transform: 'translate(' + (500 + Math.cos(a) * r).toFixed(0) + ' ' + (500 + Math.sin(a) * r).toFixed(0) + ')' }, disc);
      el('circle', { class: 'shell', r: 4 }, g);
      el('circle', { class: 'shell s2', r: 4 }, g);
      el('circle', { class: 'core', r: 2.2 }, g);
      setTimeout(function () { g.remove(); }, 6000);
      setTimeout(nova, 26000 + Math.random() * 14000);
    };
    setTimeout(nova, 9000);
  }

  // Moon phase from the mean synodic month. Good to about a day.
  var days = (Date.now() - Date.UTC(2000, 0, 6, 18, 14)) / 86400000;
  var phase = ((days % 29.530588853) + 29.530588853) % 29.530588853 / 29.530588853; // 0 new, 0.5 full
  var names = ['New moon', 'Waxing crescent', 'First quarter', 'Waxing gibbous', 'Full moon', 'Waning gibbous', 'Last quarter', 'Waning crescent'];
  var moon = document.getElementById('moon');
  el('circle', { r: 15, fill: '#1b2250', stroke: 'rgba(243,233,210,0.4)' }, moon);
  var lit = Math.cos(phase * 2 * Math.PI);            // 1 at new, -1 at full
  var rx = Math.abs(lit) * 15, waxing = phase < 0.5;
  // Lit half, then the terminator ellipse that grows or shrinks it.
  var d = 'M0 -15 A15 15 0 0 ' + (waxing ? 1 : 0) + ' 0 15 A' + rx.toFixed(2) + ' 15 0 0 ' + ((lit > 0) === waxing ? 0 : 1) + ' 0 -15Z';
  el('path', { d: d, fill: '#f3e9d2' }, moon);
  document.getElementById('moon-text').textContent = names[Math.round(phase * 8) % 8] + ', ' + Math.round((1 - lit) / 2 * 100) + '% lit';

  var readout = document.getElementById('readout');
  if (readout) readout.textContent = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) + ' local · 33.49° N 111.93° W';
})();
