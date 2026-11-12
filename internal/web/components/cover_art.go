package components

import (
	"fmt"
	"time"
)

// assetVersion changes on every start of the server or build, so browsers
// do not keep an old copy of the home page styles and script.
var assetVersion = fmt.Sprint(time.Now().Unix())

// coverAperture is the "First light" button on the home cover: a six-blade
// camera aperture that opens on the stars. Drawn in a design pass on
// 2026-11-11. Each blade is a ".blade" group, so CSS can swing it open.
// The rim is drawn as rings, not discs, and the dark ".well" behind the blades
// can fade: then the blades open on the real star chart.
const coverAperture = `<svg class="aperture" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200" aria-hidden="true" focusable="false">
  <defs>
    <path id="blade" d="M76 58.43H124C135.86 48.12 149.12 45.08 161.89 46.2A82 82 0 0 0 84.35 19.5C77 30 73 43 76 58.43Z"/>
    <path id="blade-engraving" d="M82 29C99 24 119 25 136 34M87 32C98 30 111 30 122 32"
          fill="none" stroke="#f3e9d2" stroke-width=".55" stroke-opacity=".24" stroke-linecap="round"/>
    <path id="index" d="M100 8v6M94 9v2m12-2v2" fill="none" stroke="#f3e9d2" stroke-width=".85" stroke-opacity=".6"/>
    <clipPath id="aperture-rim">
      <circle cx="100" cy="100" r="81.7"/>
    </clipPath>
  </defs>

  <circle class="well" cx="100" cy="100" r="85" fill="#04060e"/>
  <circle cx="100" cy="100" r="88.8" fill="none" stroke="#0d1230" stroke-width="14.6"/>
  <circle cx="100" cy="100" r="88.5" fill="none" stroke="#151a36" stroke-width="7"/>
  <circle cx="100" cy="100" r="96" fill="none" stroke="#f3e9d2" stroke-width="1.25" stroke-opacity=".78"/>
  <circle cx="100" cy="100" r="92" fill="none" stroke="#f3e9d2" stroke-width=".65" stroke-opacity=".34"/>
  <circle cx="100" cy="100" r="85" fill="none" stroke="#f3e9d2" stroke-width=".75" stroke-opacity=".5"/>
  <g>
    <use href="#index"/>
    <use href="#index" transform="rotate(60 100 100)"/>
    <use href="#index" transform="rotate(120 100 100)"/>
    <use href="#index" transform="rotate(180 100 100)"/>
    <use href="#index" transform="rotate(240 100 100)"/>
    <use href="#index" transform="rotate(300 100 100)"/>
  </g>

  <g class="star-field" fill="#f3e9d2">
    <path d="M85 78l.85 2.15L88 81l-2.15.85L85 84l-.85-2.15L82 81l2.15-.85Z"/>
    <path d="M118 111l.65 1.35L120 113l-1.35.65L118 115l-.65-1.35L116 113l1.35-.65Z"/>
    <circle cx="107" cy="71" r=".85" opacity=".7"/>
    <circle cx="122" cy="86" r="1.05"/>
    <circle cx="99" cy="94" r=".65" opacity=".65"/>
    <circle cx="71" cy="103" r="1.15" opacity=".75"/>
    <circle cx="82" cy="118" r=".75" opacity=".65"/>
    <circle cx="102" cy="129" r="1.2"/>
    <circle cx="133" cy="98" r=".7" opacity=".65"/>
    <circle cx="111" cy="99" r="1" fill="#f4b223"/>
    <circle cx="64" cy="70" r=".9" opacity=".75"/>
    <circle cx="91" cy="51" r=".9" opacity=".85"/>
    <circle cx="128" cy="56" r=".8" opacity=".65"/>
    <circle cx="148" cy="88" r="1.15" opacity=".85"/>
    <circle cx="145" cy="113" r=".65" opacity=".65"/>
    <circle cx="126" cy="142" r="1"/>
    <circle cx="77" cy="144" r=".75" opacity=".7"/>
    <circle cx="51" cy="111" r=".9" opacity=".85"/>
  </g>

  <g clip-path="url(#aperture-rim)" stroke="#f3e9d2" stroke-width=".95" stroke-opacity=".55" stroke-linejoin="round">
    <g class="blade"><use href="#blade" fill="#1c2247"/><use href="#blade-engraving"/></g>
    <g transform="rotate(60 100 100)"><g class="blade"><use href="#blade" fill="#151a36"/><use href="#blade-engraving"/></g></g>
    <g transform="rotate(120 100 100)"><g class="blade"><use href="#blade" fill="#0d1230"/><use href="#blade-engraving"/></g></g>
    <g transform="rotate(180 100 100)"><g class="blade"><use href="#blade" fill="#1c2247"/><use href="#blade-engraving"/></g></g>
    <g transform="rotate(240 100 100)"><g class="blade"><use href="#blade" fill="#151a36"/><use href="#blade-engraving"/></g></g>
    <g transform="rotate(300 100 100)"><g class="blade"><use href="#blade" fill="#0d1230"/><use href="#blade-engraving"/></g></g>
  </g>
  <circle cx="100" cy="100" r="81.6" fill="none" stroke="#f3e9d2" stroke-width=".75" stroke-opacity=".55"/>
  <path d="M115 189.7a91 91 0 0 0 5-.98" fill="none" stroke="#f4b223" stroke-width="1.55"/>
  <path d="M123 188a91 91 0 0 0 4.8-1.4" fill="none" stroke="#e8467c" stroke-width="1.55"/>
  <path d="M131 185.55a91 91 0 0 0 4.6-1.8" fill="none" stroke="#6b3fa0" stroke-width="1.55"/>
</svg>`
