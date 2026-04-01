(function () {
  'use strict';

  var LINE_HEIGHT_MULTIPLIER = 1.7;
  var PARAGRAPH_GAP = 24;

  function getFontString(element) {
    var style = window.getComputedStyle(element);
    var weight = style.fontWeight || '400';
    var size = style.fontSize || '16px';
    var family = style.fontFamily || 'sans-serif';
    return weight + ' ' + size + ' ' + family;
  }

  function getLineHeight(element) {
    var style = window.getComputedStyle(element);
    var lineHeight = parseFloat(style.lineHeight);
    if (isNaN(lineHeight)) {
      lineHeight = parseFloat(style.fontSize) * LINE_HEIGHT_MULTIPLIER;
    }
    return lineHeight;
  }

  function processParagraph(paragraph) {
    var text = paragraph.textContent.trim();
    if (!text) return;

    var font = getFontString(paragraph);
    var lineHeight = getLineHeight(paragraph);
    var containerWidth = paragraph.clientWidth;
    var textColor = window.getComputedStyle(paragraph).color;

    if (containerWidth <= 0) return;

    var prepared = window.Pretext.prepareWithSegments(text, font);
    var result = window.Pretext.layoutWithLines(prepared, containerWidth, lineHeight);
    var lines = result.lines;

    if (!lines || lines.length === 0) return;

    paragraph.textContent = '';
    paragraph.style.position = 'relative';
    paragraph.style.height = (lines.length * lineHeight) + 'px';
    paragraph.style.marginBottom = PARAGRAPH_GAP + 'px';

    for (var idx = 0; idx < lines.length; idx++) {
      var span = document.createElement('span');
      span.textContent = lines[idx].text;
      span.className = 'pretext-line';
      span.style.position = 'absolute';
      span.style.left = '0';
      span.style.top = (idx * lineHeight) + 'px';
      span.style.whiteSpace = 'pre';
      span.style.color = 'inherit';
      paragraph.appendChild(span);
    }
  }

  function processContainer(container) {
    if (container.hasAttribute('data-pretext-hover-done')) return;

    container.setAttribute('data-pretext-hover-original', container.innerHTML);
    container.setAttribute('data-pretext-hover-done', 'true');

    var paragraphs = container.querySelectorAll('p');
    for (var idx = 0; idx < paragraphs.length; idx++) {
      processParagraph(paragraphs[idx]);
    }
  }

  function processAllContainers() {
    var containers = document.querySelectorAll('[data-pretext-hover]');
    for (var idx = 0; idx < containers.length; idx++) {
      processContainer(containers[idx]);
    }
  }

  var resizeTimer;
  function handleResize() {
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(function () {
      var containers = document.querySelectorAll('[data-pretext-hover-done]');
      for (var idx = 0; idx < containers.length; idx++) {
        var original = containers[idx].getAttribute('data-pretext-hover-original');
        if (!original) continue;
        containers[idx].innerHTML = original;
        containers[idx].removeAttribute('data-pretext-hover-done');
      }
      processAllContainers();
    }, 300);
  }

  function init() {
    if (typeof window.Pretext === 'undefined') {
      window.addEventListener('pretext-ready', init, { once: true });
      return;
    }
    processAllContainers();
    window.addEventListener('resize', handleResize);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
