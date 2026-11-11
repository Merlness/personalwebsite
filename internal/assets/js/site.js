// Site-wide touches: the sky warms toward sunset as the reader scrolls down a
// page, and marigold petals fall on the home page for Dia de Muertos.
(function () {
  var root = document.documentElement;
  var ticking = false;
  function dusk() {
    ticking = false;
    var room = root.scrollHeight - window.innerHeight;
    root.style.setProperty('--dusk', room > 40 ? Math.min(1, window.scrollY / room).toFixed(3) : '0');
  }
  window.addEventListener('scroll', function () {
    if (!ticking) { ticking = true; requestAnimationFrame(dusk); }
  }, { passive: true });
  dusk();

  // Petals: October 27 to November 2, or any day with ?ofrenda in the address.
  var now = new Date(), month = now.getMonth() + 1, day = now.getDate();
  var season = (month === 10 && day >= 27) || (month === 11 && day <= 2);
  var calm = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  if (!calm && document.querySelector('.sc') && (season || /[?&]ofrenda\b/.test(location.search))) {
    var layer = document.createElement('div');
    layer.className = 'petals';
    layer.setAttribute('aria-hidden', 'true');
    for (var i = 0; i < 16; i++) {
      var petal = document.createElement('span');
      petal.className = 'petal';
      petal.style.left = (4 + Math.random() * 92).toFixed(1) + '%';
      petal.style.animationDuration = (16 + Math.random() * 16).toFixed(1) + 's';
      petal.style.animationDelay = (-Math.random() * 30).toFixed(1) + 's';
      petal.style.scale = (0.6 + Math.random() * 0.7).toFixed(2);
      layer.appendChild(petal);
    }
    document.body.appendChild(layer);
  }
})();
