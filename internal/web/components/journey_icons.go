package components

// Small sketch icons for the journey map. Each one is drawn in a 24 by 24
// box and sits above its stop. Classes: "l" is a line with no fill, "f" is
// filled with the accent color, "w" is water. Anything else is paper-filled.
var journeyIcons = map[string]string{
	"cactus":    `<path class="l" d="M12 21V6M12 15H8v-4M12 12h4V8"/><path class="l" d="M7 21h10"/>`,
	"log":       `<path d="M4 14h13a3 3.5 0 0 1 0 7H4a3 3.5 0 0 1 0-7z"/><ellipse class="f" cx="17" cy="17.5" rx="2.6" ry="3.5"/><path class="l" d="M7 16.5h5M6 19h4"/>`,
	"hoodoo":    `<path d="M9.500 21l1-9h3l1 9z"/><path class="f" d="M6 12q6-7 12 0z"/><path d="M17 21l.5-5h2l.5 5z"/><path class="f" d="M15.500 16q3-3.500 6 0z"/>`,
	"rock":      `<path class="f" d="M3 21l5-9 2 2 3-8 3 7 2-2 3 10z"/><path class="l" d="M2 21h20"/>`,
	"mountains": `<path d="M2 21l7-13 4 7 3-5 6 11z"/><path class="f" d="M9 8l-2.200 4 2.200-1 2 1.500z"/><path class="f" d="M16 10l-1.500 2.500 1.500-.6 1.500 1z"/><path class="l" d="M4.500 21v-2M19.500 21v-2"/><path d="M4.500 15l-1.800 4h3.600zM19.500 15l-1.800 4h3.600z"/>`,
	"canyon":    `<path d="M2 8h6.500l2.500 13h2l2.500-13H22v13H2z"/><path class="w" d="M11 21l.6-6h.8l.6 6z"/><path class="l" d="M4 12h3.500M16.500 12H20M5 16h3M16 16h3"/>`,
	"mill":      `<path d="M7 13l5-4 5 4v5H7z"/><path class="l" d="M9 18v3M15 18v3M10 14.500h4"/><path class="w" d="M3 21q3-2 6 0t6 0 6 0"/>`,
	"lake":      `<path d="M5 14l5-8 3 4 2-2 4 6z"/><path class="w" d="M3 15q3-1.500 6 0t6 0 6 0v5q-3 1.500-6 0t-6 0-6 0z"/>`,
	"tower":     `<path d="M7 21l2-13q3-2 6 0l2 13z"/><path class="l" d="M10 9.500V20M12 8.500V20M14 9.500V20M3 21h18"/>`,
	"falls":     `<path d="M2 21V6.500h9.500l1 2.500-1.500 12z"/><path class="f" d="M4 6.500l1.600-3.700 1.600 3.700zM7.500 6.500l1.300-3 1.300 3z"/><path class="w" d="M11.500 6.500q6.500 0 7.500 6v6.500h-4.500v-6q0-3.500-2.500-4z"/><path class="l" d="M16.800 12.500v5"/><path class="w" d="M10 19q6.500-3 12 0v2H10z"/><path class="l" d="M12.500 17.200q1.200-1.600 2.400 0M19.500 17.200q1.200-1.600 2.400 0"/>`,
	"badlands":  `<path d="M2 21l3-8h4l2 4 2-7h4l3 11z"/><path class="l" d="M4.200 16h5.500M12.200 14h6.300M12 18h8"/>`,
	"geyser":    `<path class="w" d="M10.500 21l1-10h1l1 10z"/><path class="l" d="M12 11q-5-2-4-7M12 11q5-2 4-7M12 11V3.500"/><path class="l" d="M5 21h14"/>`,
	"fields":    `<path d="M2 21v-3q5-6 10 0t10 0v3z"/><path class="l" d="M4 20.500q4-3.500 8 0M12 20.500q4-3.500 8 0"/><path class="f" d="M15 12.500h4v3h-4z"/><path class="l" d="M14.500 12.500l2.500-2.300 2.500 2.300"/>`,
	"seastack":  `<path d="M6 21v-9q2-4 4 0v9z"/><path d="M13.500 21v-5q1.500-3 3 0v5z"/><path class="l" d="M8 9V7.500M7 8.500h2"/><path class="w" d="M2 21q2.500-2 5 0t5 0 5 0 5 0v1H2z"/>`,
	"trees":     `<path class="l" d="M8 21V8M16 21V5"/><path class="f" d="M8 6l-3 5h6zM8 10l-4 6h8zM16 3l-3 5h6zM16 7l-4 6h8zM16 12l-4.500 6h9z"/>`,
	"volcano":   `<path d="M3 21l6-11h6l6 11z"/><path class="f" d="M9 10l1.500 3 1.500-1.500 1.500 1.500 1.500-3z"/><path class="l" d="M12 8q-2-1.500-.5-3.500t1-3"/>`,
	"wave":      `<path d="M3 21v-6h7q-2 3 2 3t2-3h7v6z"/><path class="w" d="M10 15q-2 3 2 3t2-3z"/><path class="l" d="M9 11q3-4 6 0M11 8q1-1.500 2 0"/>`,
	"crater":    `<path d="M2 20l4-10h12l4 10z"/><ellipse class="w" cx="12" cy="14.500" rx="7" ry="3"/><path d="M9.500 15l2-3 2 3z"/>`,
	"comet":     `<path class="l" d="M17 7L5 19M17 7L8 20M17 7L4 16"/><circle class="f" cx="17" cy="7" r="2.200"/>`,
	"pier":      `<path class="l" d="M3 10h18M5 10v10M10 10v10M15 10v10M20 10v10"/><path class="w" d="M2 20q3-2 6 0t6 0 6 0 2-1v2H2z"/><circle class="f" cx="18" cy="6" r="2"/>`,
	"joshua":    `<path class="l" d="M12 21v-8M12 15l-5-3V9M12 13l5-3V6M12 13V7"/><circle class="f" cx="7" cy="8" r="1.800"/><circle class="f" cx="17" cy="5" r="1.800"/><circle class="f" cx="12" cy="6" r="1.800"/><path class="l" d="M7 21h10"/>`,
}

func journeyIcon(name string) string {
	return journeyIcons[name]
}
