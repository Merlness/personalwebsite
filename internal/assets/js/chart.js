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
  // First light. The chart stays hidden until the aperture sits on it:
  // the blades close, the aperture travels to the middle of the chart and
  // grows to the size of its ring, and only then do the blades open on the
  // real chart. The rest of the page fades in around it.
  var lens = document.querySelector('.sc-cover .lens'), chartSvg = document.querySelector('.sc .chart');
  var timers = [], steps = ['sc-going', 'sc-closing', 'sc-hole', 'sc-open'];
  function settle() {
    body.classList.add('sc-seen');
    steps.forEach(function (c) { body.classList.remove(c); });
    if (lens && lens.getAnimations) lens.getAnimations().forEach(function (a) { a.cancel(); });
    chartStage.focus({ preventScroll: true });
  }
  function open() {
    if (body.classList.contains('sc-going') || body.classList.contains('sc-seen')) return;
    try { sessionStorage.setItem('firstLight', '1'); } catch (e) {}
    if (calm || !lens || !lens.animate || !chartSvg) { settle(); return; }
    body.classList.add('sc-going', 'sc-closing');
    timers = [
      setTimeout(function () {
        var l = lens.getBoundingClientRect(), c = chartSvg.getBoundingClientRect();
        // The outer rim is 96% of the aperture; the outer ring is 94% of the chart.
        var scale = (c.width * 0.94) / (l.width * 0.96);
        var cx = c.left + c.width / 2, cy = c.top + c.height / 2;
        // The cover gets a round hole exactly under the aperture rim.
        body.style.setProperty('--hx', cx + 'px');
        body.style.setProperty('--hy', cy + 'px');
        body.style.setProperty('--hr', (c.width * 0.47 - 4) + 'px');
        lens.animate([{ transform: 'none' }, { transform: 'translate(' + (cx - l.left - l.width / 2) + 'px,' + (cy - l.top - l.height / 2) + 'px) scale(' + scale + ')' }],
          { duration: 620, easing: 'cubic-bezier(0.4, 0, 0.2, 1)', fill: 'forwards' });
      }, 220),
      setTimeout(function () {
        body.classList.remove('sc-closing');
        body.classList.add('sc-hole', 'sc-open');
        lens.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 420, delay: 380, easing: 'linear', fill: 'forwards' });
      }, 880),
      setTimeout(settle, 1720)
    ];
  }
  if (openBtn) openBtn.addEventListener('click', open);

  // The name in the header goes back to the beginning: the cover closes again.
  function showCover() {
    try { sessionStorage.removeItem('firstLight'); } catch (e) {}
    timers.forEach(clearTimeout);
    if (lens && lens.getAnimations) lens.getAnimations().forEach(function (a) { a.cancel(); });
    body.classList.remove('sc-seen');
    steps.forEach(function (c) { body.classList.remove(c); });
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
    { id: 'about', dive: [626, 360], title: 'About', sub: 'el observador', href: '/about', go: 'Meet the observer →',
      text: 'Physics teacher turned software craftsman.',
      img: '/assets/aboutme/profile_w600.jpg',
      data: [['Observer', 'Merl Martin'], ['Station', 'Scottsdale, Arizona'], ['Lat, Lon', '33.49° N, 111.93° W']],
      stars: [[452, 512], [548, 428], [560, 442], [464, 526], [500, 486], [470, 560], [500, 568], [530, 560]],
      lines: [[0, 1, 2, 3, 0], [4, 5], [4, 6], [4, 7]], lead: 1, label: [468, 615] }
  ];

  // Dive in: fly the chart into the clicked shape, then open its page.
  var chart = document.querySelector('.sc .chart'), wrap = document.querySelector('.sc');
  var calm = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  function leave(part) {
    if (part.href.charAt(0) === '/') { try { sessionStorage.setItem('arrive', part.id); } catch (e) {} }
    window.location.href = part.href;
  }
  // How long the short action runs before the dive starts.
  var waits = { journeys: 950, journal: 460, about: 480 };
  // How long the carried thing flies. The Journal rule takes its time.
  var flights = { journal: 1250 };
  function dive(e, part) {
    if (calm || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button) return; // plain link
    e.preventDefault();
    if (wrap.classList.contains('acting') || wrap.classList.contains('diving')) return;
    // First a short action in the shape itself, then the dive.
    var link = e.currentTarget, act = el('g', { class: 'act act-' + part.id }, link);
    (acts[part.id] || function () {})(act, part);
    wrap.classList.add('acting');
    // Photography does not dive. The shutter closes, and one flash grows out
    // of the lens until it covers the screen. The portfolio fades in from it.
    if (part.id === 'photo') {
      setTimeout(function () {
        var c = chartSvg.getBoundingClientRect(), flash = document.createElement('div');
        flash.className = 'sc-flash';
        flash.style.setProperty('--fx', (c.left + part.dive[0] / 1000 * c.width) + 'px');
        flash.style.setProperty('--fy', (c.top + part.dive[1] / 1000 * c.height) + 'px');
        document.body.appendChild(flash);
      }, 210);
      setTimeout(function () { leave(part); }, 620);
      return;
    }
    // Tech keeps the long dive. The others use a shorter, smoother one, and
    // the zoom, the fade, and the carried thing all start together.
    var smooth = part.id !== 'tech', wait = waits[part.id] || 520;
    setTimeout(function () {
      if (smooth) { wrap.classList.add('smooth'); body.classList.add('sc-smooth'); }
      if (part.id === 'about') body.classList.add('sc-wash');
      if (carries[part.id]) carries[part.id](link, act);
      chart.style.transformOrigin = (part.dive[0] / 10) + '% ' + (part.dive[1] / 10) + '%';
      wrap.classList.add('diving');
      body.classList.add('sc-diving');
    }, wait);
    setTimeout(function () { leave(part); }, wait + (smooth ? (flights[part.id] || 740) + 40 : 930));
  }

  // One thing from the chart comes along to the next page, and it flies in a
  // straight line to the exact place it has there. To know that place, the
  // next page is loaded ahead of time in a hidden frame the size of the
  // window, and the target is measured in it. The next page then shows the
  // target above its fade-in (see html[data-arrive] in input.css), so the
  // thing never stops or jumps.
  var frames = {}, targets = { journeys: 'main h1', journal: 'main .h-1.w-24' };
  if (!calm) {
    window.addEventListener('load', function () {
      setTimeout(function () {
        parts.forEach(function (part) {
          if (!targets[part.id]) return;
          var f = document.createElement('iframe');
          f.className = 'sc-ahead';
          f.setAttribute('aria-hidden', 'true');
          f.setAttribute('tabindex', '-1');
          f.src = part.href;
          document.body.appendChild(f);
          frames[part.id] = f;
        });
      }, 1200);
    });
  }
  function place(id) {
    try {
      var doc = frames[id].contentDocument, t = doc.readyState === 'complete' && doc.querySelector(targets[id]);
      if (!t) return null;
      var r = t.getBoundingClientRect();
      return r.width ? { rect: r, style: frames[id].contentWindow.getComputedStyle(t) } : null;
    } catch (e) { return null; }
  }
  var hidden = [];
  function carry(node, from, to, turn, fadeIn, duration) {
    node.classList.add('sc-carry');
    node.style.left = to.left + 'px'; node.style.top = to.top + 'px';
    node.style.width = to.width + 'px'; node.style.height = to.height + 'px';
    document.body.appendChild(node);
    var k = from.width / to.width;
    var dx = from.left + from.width / 2 - to.left - to.width / 2, dy = from.top + from.height / 2 - to.top - to.height / 2;
    var frames = [{ transform: 'translate(' + dx + 'px,' + dy + 'px) scale(' + k + ') rotate(' + (turn || 0) + 'deg)', opacity: fadeIn ? 0 : 1 }];
    if (duration) {
      // The long flight: the thing swings out in an arc, grows past its
      // final size as it unfurls, and settles into place.
      frames.push({ opacity: 1, offset: 0.2 });
      frames.push({ transform: 'translate(' + (dx * 0.42 - dy * 0.16) + 'px,' + (dy * 0.42 + dx * 0.16) + 'px) scale(1.3) rotate(' + (-(turn || 0) * 0.5) + 'deg)', offset: 0.62 });
      frames.push({ transform: 'translate(0, 0) scale(1.08) rotate(0deg)', offset: 0.9 });
    } else {
      frames.push({ opacity: 1, offset: 0.3 });
    }
    frames.push({ transform: 'none', opacity: 1 });
    node.animate(frames, { duration: duration || 740, easing: duration ? 'cubic-bezier(0.45, 0, 0.3, 1)' : 'cubic-bezier(0.4, 0, 0.2, 1)', fill: 'both' });
    return k;
  }
  function hide(node) { node.style.visibility = 'hidden'; hidden.push(node); }
  var carries = {
    // The word "Journeys" leaves the chart and becomes the title of the page.
    journeys: function (link) {
      var to = place('journeys');
      if (!to) return;
      var label = link.querySelector('text'), from = label.getBoundingClientRect();
      var node = document.createElement('div');
      node.className = 'sc-word';
      node.style.fontSize = to.style.fontSize;
      node.style.lineHeight = to.rect.height + 'px';
      node.innerHTML = '<span class="to">Journeys</span><span class="from">Journeys</span>';
      // The title is as wide as its column; the word in it is narrower.
      var probe = document.createElement('span');
      probe.className = 'sc-word'; probe.style.cssText = 'position:fixed;visibility:hidden;font-size:' + to.style.fontSize;
      probe.textContent = 'Journeys'; document.body.appendChild(probe);
      var wordWidth = probe.getBoundingClientRect().width; probe.remove();
      var box = { left: to.rect.left + (to.rect.width - wordWidth) / 2, top: to.rect.top, width: wordWidth, height: to.rect.height };
      var k = carry(node, from, box);
      node.querySelector('.from').style.fontSize = (15 * chartSvg.getBoundingClientRect().width / 1000 / k) + 'px';
      hide(label);
    },
    // One line of the notebook becomes the rule under the Journal title.
    journal: function (link) {
      var to = place('journal');
      if (!to) return;
      var lines = link.querySelectorAll('line'), line = lines[lines.length - 1], from = line.getBoundingClientRect();
      var node = document.createElement('div');
      node.className = 'h-1 w-24';
      // The line in the notebook slopes up to the right by about 15 degrees.
      carry(node, from, to.rect, -14.6, true, flights.journal);
      line.style.transition = 'opacity 0.25s'; line.style.opacity = 0;
      hidden.push(line);
    }
  };

  function pts(part, list) { return list.map(function (i) { return part.stars[i].join(','); }).join(' '); }
  var acts = {
    // The shutter: the lens closes. The flash is the whole screen (see dive).
    photo: function (g, part) {
      el('polygon', { class: 'shutter', points: pts(part, [8, 9, 10, 11, 12, 13]) }, g);
    },
    // A climb: a jagged marigold trail works its way up the ridge, switchback
    // by switchback, and a flag goes up on top.
    journeys: function (g, part) {
      var ridge = [0, 1, 2, 3].map(function (i) { return part.stars[i]; }), trail = [], side = 1;
      for (var n = 1; n < ridge.length; n++) {
        var a = ridge[n - 1], b = ridge[n], dx = b[0] - a[0], dy = b[1] - a[1], len = Math.sqrt(dx * dx + dy * dy);
        var count = Math.max(2, Math.round(len / 15));
        trail.push(a.join(','));
        for (var c = 1; c < count; c++) {
          side = -side;
          trail.push((a[0] + dx * c / count - dy / len * 7 * side).toFixed(1) + ',' + (a[1] + dy * c / count + dx / len * 7 * side).toFixed(1));
        }
      }
      trail.push(ridge[ridge.length - 1].join(','));
      el('polyline', { class: 'climb glow', points: trail.join(' '), pathLength: 100 }, g);
      el('polyline', { class: 'climb', points: trail.join(' '), pathLength: 100 }, g);
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
    wrap.classList.remove('diving', 'acting', 'smooth');
    body.classList.remove('sc-diving', 'sc-smooth', 'sc-wash');
    Array.prototype.forEach.call(document.querySelectorAll('.sc .act, .sc-carry, .sc-flash'), function (n) { n.remove(); });
    hidden.forEach(function (n) { n.style.visibility = ''; n.style.opacity = ''; });
    hidden = [];
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
