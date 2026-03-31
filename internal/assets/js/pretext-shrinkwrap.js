/**
 * Pretext Shrinkwrap Titles
 *
 * Uses Pretext's walkLineRanges() to find the tightest possible
 * container width for text blocks, then applies that width so
 * titles and card labels fit their content precisely with no
 * awkward whitespace.
 *
 * Hooks into elements with [data-pretext-shrinkwrap].
 */
(function () {
  'use strict';

  

  /**
   * Compute the font string from an element's computed styles.
   */
  function getFontString(el) {
    var style = window.getComputedStyle(el);
    var weight = style.fontWeight || '400';
    var size = style.fontSize || '16px';
    var family = style.fontFamily || 'Inter, sans-serif';
    return weight + ' ' + size + ' ' + family;
  }

  /**
   * Find the minimum width that still produces the same line count
   * as the current container width. This is the "shrinkwrap" width.
   */
  function findShrinkwrapWidth(text, font, maxWidth) {
    var prepared = window.Pretext.prepareWithSegments(text, font);

    // First, find the line count at current width
    var baseLineCount = 0;
    window.Pretext.walkLineRanges(prepared, maxWidth, function () {
      baseLineCount++;
    });

    if (baseLineCount <= 1) {
      // Single line: find exact width needed
      var maxLineWidth = 0;
      window.Pretext.walkLineRanges(prepared, maxWidth, function (start, end, width) {
        if (width > maxLineWidth) maxLineWidth = width;
      });
      return Math.ceil(maxLineWidth) + 2; // +2 for subpixel safety
    }

    // Multi-line: binary search for the narrowest width that keeps
    // the same line count
    var lo = 40; // minimum reasonable width
    var hi = maxWidth;
    var best = maxWidth;

    while (lo <= hi) {
      var mid = Math.floor((lo + hi) / 2);
      var count = 0;
      window.Pretext.walkLineRanges(prepared, mid, function () {
        count++;
      });

      if (count <= baseLineCount) {
        best = mid;
        hi = mid - 1;
      } else {
        lo = mid + 1;
      }
    }

    return best;
  }

  /**
   * Process a single [data-pretext-shrinkwrap] element.
   */
  function processShrinkwrap(el) {
    var text = el.textContent.trim();
    if (!text) return;

    var font = getFontString(el);
    var maxWidth = el.parentElement ? el.parentElement.clientWidth : el.clientWidth;

    if (maxWidth <= 0) return;

    var optimalWidth = findShrinkwrapWidth(text, font, maxWidth);

    // Only apply if we actually shrink by a meaningful amount (>10px)
    if (maxWidth - optimalWidth > 10) {
      el.style.maxWidth = optimalWidth + 'px';
      // Center the shrinkwrapped element if it was centered before
      var style = window.getComputedStyle(el);
      if (style.textAlign === 'center' || style.marginLeft === 'auto') {
        el.style.marginLeft = 'auto';
        el.style.marginRight = 'auto';
      }
    }
  }

  /**
   * Initialize all [data-pretext-shrinkwrap] elements.
   */
  function init() {
    var elements = document.querySelectorAll('[data-pretext-shrinkwrap]');
    for (var i = 0; i < elements.length; i++) {
      processShrinkwrap(elements[i]);
    }
  }

  // Debounced resize handler
  var resizeTimer;
  function handleResize() {
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(function () {
      var elements = document.querySelectorAll('[data-pretext-shrinkwrap]');
      for (var i = 0; i < elements.length; i++) {
        // Reset before recalculating
        elements[i].style.maxWidth = '';
        processShrinkwrap(elements[i]);
      }
    }, 250);
  }

  function start() {
    if (typeof window.Pretext === 'undefined') {
      window.addEventListener('pretext-ready', start, { once: true });
      return;
    }
    init();
    window.addEventListener('resize', handleResize);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', start);
  } else {
    start();
  }
})();
