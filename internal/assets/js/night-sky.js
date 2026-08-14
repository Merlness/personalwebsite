// Spicy-mode night sky: drag the home hero to pan the starfield.
// Easing is handled by a CSS transition on .skyfield, so there is no
// animation loop to stall when the tab is backgrounded.
(function () {
  const hero = document.getElementById('home-hero');
  const field = document.querySelector('.constellations .skyfield');
  if (!hero || !field) return;

  const MAX_X = 150;
  const MAX_Y = 100;
  let x = 0, y = 0;
  let startX = 0, startY = 0, originX = 0, originY = 0;
  let dragging = false;

  const clamp = (v, m) => Math.max(-m, Math.min(m, v));
  const spicy = () => document.documentElement.getAttribute('data-theme') === 'rhcp';

  function apply() {
    field.style.transform = 'translate(' + x.toFixed(2) + 'px, ' + y.toFixed(2) + 'px)';
  }

  hero.addEventListener('pointerdown', (e) => {
    if (!spicy()) return;
    if (e.target.closest('a, button')) return;
    dragging = true;
    startX = e.clientX;
    startY = e.clientY;
    originX = x;
    originY = y;
    hero.classList.add('sky-dragging');
    try {
      hero.setPointerCapture(e.pointerId);
    } catch (err) {
      // pointer capture is best-effort
    }
  });

  hero.addEventListener('pointermove', (e) => {
    if (!dragging) return;
    x = clamp(originX + (e.clientX - startX) * 0.6, MAX_X);
    y = clamp(originY + (e.clientY - startY) * 0.6, MAX_Y);
    apply();
  });

  ['pointerup', 'pointercancel'].forEach((ev) =>
    hero.addEventListener(ev, () => {
      dragging = false;
      hero.classList.remove('sky-dragging');
    })
  );
})();
