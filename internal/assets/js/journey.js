// Journey page: draws the route, moves the map, and counts the numbers as
// the reader scrolls through the stops. No dependencies.
(function () {
  var root = document.querySelector('.jr');
  if (!root) return;

  var stops = Array.prototype.slice.call(root.querySelectorAll('.jr-stop'));
  var dots = Array.prototype.slice.call(root.querySelectorAll('.jr-dot'));
  var route = root.querySelector('.jr-route');
  var cam = root.querySelector('.jr-cam');
  var pen = root.querySelector('.jr-pen');
  var svg = root.querySelector('.jr-map svg');
  var end = root.querySelector('.jr-end');
  if (!stops.length || !route || !cam || !svg) return;

  var out = {
    miles: root.querySelector('[data-jr="miles"]'),
    elev: root.querySelector('[data-jr="elev"]'),
    day: root.querySelector('[data-jr="day"]')
  };

  var data = stops.map(function (el) {
    return {
      el: el,
      len: parseFloat(el.dataset.len) || 0,
      miles: parseFloat(el.dataset.miles) || 0,
      elev: parseFloat(el.dataset.elev) || 0,
      day: el.dataset.day || '1',
      scene: el.dataset.scene || '',
      paper: el.dataset.paper || '',
      ink: el.dataset.ink || '',
      accent: el.dataset.accent || ''
    };
  });

  var total = route.getTotalLength();
  var dataTotal = parseFloat(root.dataset.routeLen) || total;
  var k = total / dataTotal; // data lengths are straight-line sums; keep them in step with the real path
  var box = svg.viewBox.baseVal;
  var ZOOM = 2.6;

  route.style.strokeDasharray = total;
  route.style.strokeDashoffset = total;

  function lerp(a, b, t) { return a + (b - a) * t; }
  function fmt(n) { return Math.round(n).toLocaleString('en-US'); }

  // The page starts in the trip's own colors and takes each stop's colors as
  // the reader reaches it. CSS blends between them.
  var home = {
    paper: root.style.getPropertyValue('--jr-paper'),
    ink: root.style.getPropertyValue('--jr-ink'),
    accent: root.style.getPropertyValue('--jr-accent')
  };
  function paint(theme) {
    ['paper', 'ink', 'accent'].forEach(function (name) {
      root.style.setProperty('--jr-' + name, theme[name] || home[name]);
    });
  }

  var lastActive = '';
  var ticking = false;

  function update() {
    ticking = false;
    var line = window.innerHeight * (window.innerWidth <= 860 ? 0.72 : 0.55);

    // Which stop has crossed the reading line most recently?
    var active = -1;
    for (var i = 0; i < data.length; i++) {
      if (data[i].el.getBoundingClientRect().top <= line) active = i; else break;
    }
    var finished = end && end.getBoundingClientRect().top <= line;

    var len = 0, miles = 0, elev = data[0].elev, day = data[0].day;
    var near = active; // the farthest stop whose icon shows
    if (active >= 0) {
      var cur = data[active];
      var next = data[active + 1];
      var t = 0;
      if (next) {
        var a = cur.el.getBoundingClientRect().top;
        var b = next.el.getBoundingClientRect().top;
        // Hold on the stop for the first 45% of its section, then drive.
        t = Math.min(1, Math.max(0, ((line - a) / (b - a) - 0.45) / 0.55));
        if (t > 0.3) near = active + 1; // close enough to see it coming
        len = lerp(cur.len, next.len, t);
        miles = lerp(cur.miles, next.miles, t);
        elev = lerp(cur.elev, next.elev, t);
      } else {
        len = cur.len; miles = cur.miles; elev = cur.elev;
      }
      day = cur.day;
    }

    var drawn = len * k;
    route.style.strokeDashoffset = Math.max(0, total - drawn);
    var p = route.getPointAtLength(drawn);
    pen.setAttribute('cx', p.x);
    pen.setAttribute('cy', p.y);

    if (active < 0 || finished) {
      cam.style.transform = 'none';
      pen.style.opacity = active < 0 ? 0 : 1;
    } else {
      var tx = box.width / 2 - ZOOM * p.x;
      var ty = box.height / 2 - ZOOM * p.y;
      cam.style.transform = 'translate(' + tx.toFixed(1) + 'px,' + ty.toFixed(1) + 'px) scale(' + ZOOM + ')';
      pen.style.opacity = 1;
    }

    if (out.miles) out.miles.textContent = fmt(Math.round(miles / 10) * 10);
    if (out.elev) out.elev.textContent = fmt(Math.round(elev / 10) * 10);
    if (out.day) out.day.textContent = day;

    if (finished) near = data.length;
    var key = (finished ? -3 : active) + ':' + near;
    if (key !== lastActive) {
      lastActive = key;
      data.forEach(function (d, i) { d.el.classList.toggle('is-active', i === active); });
      dots.forEach(function (dot, i) {
        dot.classList.toggle('is-active', i === active && !finished);
        dot.classList.toggle('is-done', i <= active);
        dot.classList.toggle('is-seen', i <= near);
      });
      paint(active >= 0 && !finished ? data[active] : home);
      root.classList.toggle('jr-night', active >= 0 && data[active].scene === 'night');
    }
  }

  function onScroll() {
    if (!ticking) { ticking = true; requestAnimationFrame(update); }
  }

  window.addEventListener('scroll', onScroll, { passive: true });
  window.addEventListener('resize', onScroll);
  update();
})();
