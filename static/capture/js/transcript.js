// Assembles final speech-recognition results into one transcript.
// Android Chrome reports cumulative finals (each contains the whole
// transcript so far); desktop reports distinct segments. Detect which
// by checking if a new final extends the accumulated text.

export function assembleFinals(results) {
  let finals = "";
  for (let i = 0; i < results.length; i++) {
    if (!results[i].isFinal) continue;
    const t = results[i][0].transcript.trim();
    if (!t) continue;
    if (t.toLowerCase().startsWith(finals.toLowerCase())) finals = t;
    else if (finals.toLowerCase().endsWith(t.toLowerCase())) continue;
    else finals += (finals ? " " : "") + t;
  }
  return finals;
}
