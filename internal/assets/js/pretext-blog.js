/**
 * Pretext Blog Magazine Layout
 *
 * Uses Pretext's layoutNextLine() with variable widths to flow
 * paragraph text around inset images, creating a magazine-style
 * reading experience for blog posts.
 *
 * Hooks into elements with [data-pretext-flow] and looks for
 * <img> children with [data-pretext-inset="left"|"right"].
 */
(function () {
  'use strict';

  

  var INSET_MARGIN = 20; // gap between image and text in px
  var LINE_HEIGHT_MULTIPLIER = 1.7;

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
   * Get the line height in pixels from an element.
   */
  function getLineHeight(el) {
    var style = window.getComputedStyle(el);
    var lh = parseFloat(style.lineHeight);
    if (isNaN(lh)) {
      lh = parseFloat(style.fontSize) * LINE_HEIGHT_MULTIPLIER;
    }
    return lh;
  }

  /**
   * Layout a single paragraph with optional inset image.
   *
   * Returns an array of { text, x, y, width } line objects that
   * the caller can render into a container.
   */
  function layoutParagraph(text, font, containerWidth, lineHeight, inset) {
    var prepared = window.Pretext.prepareWithSegments(text, font);
    var lines = [];
    var cursor = { segmentIndex: 0, graphemeIndex: 0 };
    var y = 0;

    while (true) {
      var availableWidth = containerWidth;
      var xOffset = 0;

      // If an inset image occupies this vertical region, shrink the line
      if (inset && y < inset.bottom && (y + lineHeight) > inset.top) {
        availableWidth = containerWidth - inset.width - INSET_MARGIN;
        if (inset.side === 'right') {
          // Text stays on the left, no x offset
          xOffset = 0;
        } else {
          // Image on left, text shifts right
          xOffset = inset.width + INSET_MARGIN;
        }
      }

      // Guard against impossibly narrow lines
      if (availableWidth < 40) {
        availableWidth = containerWidth;
        xOffset = 0;
      }

      var line = window.Pretext.layoutNextLine(prepared, cursor, availableWidth);
      if (line === null) break;

      lines.push({
        text: line.text,
        x: xOffset,
        y: y,
        width: line.width
      });

      cursor = line.end;
      y += lineHeight;
    }

    return lines;
  }

  /**
   * Render laid-out lines into a container element.
   */
  function renderLines(container, lines, lineHeight, textColor) {
    var wrapper = document.createElement('div');
    wrapper.style.position = 'relative';
    wrapper.style.height = (lines.length * lineHeight) + 'px';

    for (var i = 0; i < lines.length; i++) {
      var span = document.createElement('span');
      span.textContent = lines[i].text;
      span.className = 'pretext-line';
      span.style.position = 'absolute';
      span.style.left = lines[i].x + 'px';
      span.style.top = lines[i].y + 'px';
      span.style.whiteSpace = 'nowrap';
      span.style.color = textColor;
      wrapper.appendChild(span);
    }

    container.appendChild(wrapper);
  }

  /**
   * Process a [data-pretext-flow] element.
   * Finds paragraphs and inset images, then relayouts text around images.
   */
  function processFlowContainer(container) {
    var font = getFontString(container);
    var lineHeight = getLineHeight(container);
    var containerWidth = container.clientWidth;
    var textColor = window.getComputedStyle(container).color;

    // Find all direct children: paragraphs and images
    var children = container.children;
    var insetImages = [];

    // Gather inset image data
    for (var i = 0; i < children.length; i++) {
      var child = children[i];
      if (child.tagName === 'IMG' && child.hasAttribute('data-pretext-inset')) {
        insetImages.push({
          element: child,
          side: child.getAttribute('data-pretext-inset') || 'right',
          width: child.offsetWidth || 300,
          height: child.offsetHeight || 200
        });
      }
    }

    // If no inset images, nothing special to do (let CSS handle it)
    if (insetImages.length === 0) return;

    // Process each paragraph that comes after an inset image
    var accumulatedHeight = 0;

    for (var j = 0; j < children.length; j++) {
      var el = children[j];

      if (el.tagName === 'IMG' && el.hasAttribute('data-pretext-inset')) {
        var imgData = null;
        for (var k = 0; k < insetImages.length; k++) {
          if (insetImages[k].element === el) {
            imgData = insetImages[k];
            break;
          }
        }
        if (imgData) {
          imgData.topOffset = accumulatedHeight;
        }
        // Style the image to float
        el.style.cssFloat = imgData.side;
        el.style.position = 'relative';
        if (imgData.side === 'right') {
          el.style.marginLeft = INSET_MARGIN + 'px';
          el.style.marginRight = '0';
        } else {
          el.style.marginRight = INSET_MARGIN + 'px';
          el.style.marginLeft = '0';
        }
        el.style.marginBottom = INSET_MARGIN + 'px';
        el.style.maxWidth = '45%';
        continue;
      }

      if (el.tagName === 'P' || el.classList.contains('pretext-paragraph')) {
        var text = el.textContent.trim();
        if (!text) continue;

        // Find the nearest preceding inset image
        var activeInset = null;
        for (var m = 0; m < insetImages.length; m++) {
          var img = insetImages[m];
          var imgBottom = (img.topOffset || 0) + img.height;
          if (accumulatedHeight < imgBottom) {
            activeInset = {
              top: (img.topOffset || 0) - accumulatedHeight,
              bottom: imgBottom - accumulatedHeight,
              width: img.width,
              side: img.side
            };
            break;
          }
        }

        if (activeInset && activeInset.top < 0) {
          activeInset.bottom = activeInset.bottom;
          activeInset.top = 0;
        }

        var lines = layoutParagraph(text, font, containerWidth, lineHeight, activeInset);

        el.textContent = '';
        el.style.position = 'relative';
        el.style.minHeight = (lines.length * lineHeight) + 'px';
        el.style.marginBottom = '24px';
        renderLines(el, lines, lineHeight, textColor);

        accumulatedHeight += lines.length * lineHeight + parseFloat(window.getComputedStyle(el).marginBottom || 0);
      }
    }
  }

  /**
   * Initialize all [data-pretext-flow] containers on the page.
   */
  function init() {
    var containers = document.querySelectorAll('[data-pretext-flow]');
    for (var i = 0; i < containers.length; i++) {
      processFlowContainer(containers[i]);
    }
  }

  // Run on page load and on resize (debounced)
  var resizeTimer;
  function handleResize() {
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(function () {
      // Re-process: reset containers first
      var containers = document.querySelectorAll('[data-pretext-flow]');
      for (var i = 0; i < containers.length; i++) {
        // We need to restore original content before re-laying out.
        // Store original content on first run.
        if (!containers[i].hasAttribute('data-pretext-initialized')) continue;
        var original = containers[i].getAttribute('data-pretext-original');
        if (original) {
          containers[i].innerHTML = original;
          processFlowContainer(containers[i]);
        }
      }
    }, 250);
  }

  // Store original HTML before first processing
  function initWithStorage() {
    var containers = document.querySelectorAll('[data-pretext-flow]');
    for (var i = 0; i < containers.length; i++) {
      if (!containers[i].hasAttribute('data-pretext-initialized')) {
        containers[i].setAttribute('data-pretext-original', containers[i].innerHTML);
        containers[i].setAttribute('data-pretext-initialized', 'true');
        processFlowContainer(containers[i]);
      }
    }
  }

  function init() {
    if (typeof window.Pretext === 'undefined') {
      window.addEventListener('pretext-ready', init, { once: true });
      return;
    }
    initWithStorage();
    window.addEventListener('resize', handleResize);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
