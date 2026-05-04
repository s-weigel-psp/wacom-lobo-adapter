// extension/test-coords.js — development verification only, not in manifest
// Run: node test-coords.js
// Tests the getPhysicalCoords() formula against docs/protocol.md Section 3 examples.

// --- GREEN phase: full implementation ---
function getPhysicalCoords(element) {
  const rect = element.getBoundingClientRect();
  const dpr  = window.devicePixelRatio;
  return {
    x:      Math.round((rect.left   + window.screenX) * dpr),
    y:      Math.round((rect.top    + window.screenY) * dpr),
    width:  Math.round(rect.width   * dpr),
    height: Math.round(rect.height  * dpr),
  };
}

// Mock browser globals
global.window = { devicePixelRatio: 1.5, screenX: 0, screenY: 0 };

let passed = 0;
let failed = 0;

function assert(label, fn, expected) {
  try {
    const actual = fn();
    const ok = JSON.stringify(actual) === JSON.stringify(expected);
    console.log(ok ? 'PASS' : 'FAIL', label);
    if (!ok) {
      console.log('  expected:', expected);
      console.log('  actual:  ', actual);
      failed++;
    } else {
      passed++;
    }
  } catch (err) {
    console.log('FAIL', label, '-', err.message);
    failed++;
  }
}

// Test 1: 150% DPI — protocol.md Section 3 example
window.devicePixelRatio = 1.5;
window.screenX = 0; window.screenY = 0;
assert(
  '150% DPI, window@(0,0)',
  () => getPhysicalCoords({ getBoundingClientRect: () => ({ left: 160, top: 180, width: 960, height: 360 }) }),
  { x: 240, y: 270, width: 1440, height: 540 }
);

// Test 2: 100% DPI
window.devicePixelRatio = 1.0;
window.screenX = 0; window.screenY = 0;
assert(
  '100% DPI, window@(0,0)',
  () => getPhysicalCoords({ getBoundingClientRect: () => ({ left: 100, top: 200, width: 800, height: 400 }) }),
  { x: 100, y: 200, width: 800, height: 400 }
);

// Test 3: 125% DPI, window offset
window.devicePixelRatio = 1.25;
window.screenX = 100; window.screenY = 50;
assert(
  '125% DPI, window@(100,50)',
  () => getPhysicalCoords({ getBoundingClientRect: () => ({ left: 0, top: 0, width: 960, height: 540 }) }),
  { x: 125, y: 63, width: 1200, height: 675 }
);

console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
