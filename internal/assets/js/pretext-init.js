import {
  prepare,
  prepareWithSegments,
  layout,
  layoutWithLines,
  layoutNextLine,
  walkLineRanges,
  clearCache,
  setLocale
} from './pretext-bundle.js';

window.Pretext = {
  prepare: prepare,
  prepareWithSegments: prepareWithSegments,
  layout: layout,
  layoutWithLines: layoutWithLines,
  layoutNextLine: layoutNextLine,
  walkLineRanges: walkLineRanges,
  clearCache: clearCache,
  setLocale: setLocale
};

window.dispatchEvent(new Event('pretext-ready'));
