import fs from "fs";
import path from "path";

function median(nums) {
  if (!nums.length) return null;
  const a = [...nums].sort((x, y) => x - y);
  const m = Math.floor(a.length / 2);
  return a.length % 2 ? a[m] : (a[m - 1] + a[m]) / 2;
}

function tol(v, ratio = 0.15, floor = 1) {
  if (v == null) return null;
  return Math.max(floor, Math.abs(v) * ratio);
}

function parseArg(flag) {
  const i = process.argv.indexOf(flag);
  if (i >= 0 && i + 1 < process.argv.length) return process.argv[i + 1];
  return null;
}

function readInputJSON() {
  if (!process.stdin.isTTY) {
    const raw = fs.readFileSync(0, "utf-8").trim();
    if (raw) return JSON.parse(raw);
  }
  return null;
}

const input = readInputJSON() || {};
let templateMetrics = input.template_metrics || null;
if (!templateMetrics) {
  const templatePath = input.template_metrics_path || parseArg("--template");
  if (!templatePath) {
    console.error("missing template_metrics / --template");
    process.exit(1);
  }
  templateMetrics = JSON.parse(fs.readFileSync(templatePath, "utf-8"));
}

let rulesText = input.rules_text || "";
if (!rulesText) {
  const rulesFile = parseArg("--rules-file");
  if (rulesFile) rulesText = fs.readFileSync(rulesFile, "utf-8");
}

const allRuns = (templateMetrics.slides || []).flatMap((s) => s.textRuns || []);
const allShapes = (templateMetrics.slides || []).flatMap((s) => s.shapes || []);

const fontSizes = allRuns.map((r) => r.fontSizePt).filter((v) => typeof v === "number" && Number.isFinite(v));
const firstRuns = (templateMetrics.slides || []).map((s) => (s.textRuns || [])[0]).filter(Boolean);
const firstX = firstRuns.map((r) => r.box?.x).filter((v) => typeof v === "number");
const firstY = firstRuns.map((r) => r.box?.y).filter((v) => typeof v === "number");

const latinFreq = new Map();
const eaFreq = new Map();
for (const r of allRuns) {
  const latin = r.fontName?.latin;
  const ea = r.fontName?.ea;
  if (latin) latinFreq.set(latin, (latinFreq.get(latin) || 0) + 1);
  if (ea) eaFreq.set(ea, (eaFreq.get(ea) || 0) + 1);
}

function topFonts(m, n = 3) {
  return [...m.entries()].sort((a, b) => b[1] - a[1]).slice(0, n).map((x) => x[0]);
}

const fontMedian = median(fontSizes);
const firstXMedian = median(firstX);
const firstYMedian = median(firstY);

const hardRules = (rulesText || "")
  .split(/\r?\n/)
  .map((x) => x.trim())
  .filter((x) => x && !x.toLowerCase().startsWith("rules_text:"));

const profile = {
  version: "v1",
  generatedAt: new Date().toISOString(),
  source: templateMetrics.sourcePptx || null,
  stats: {
    slideCount: templateMetrics.slideCount || (templateMetrics.slides || []).length,
    fontSizePtMedian: fontMedian,
    firstTextBoxXMedian: firstXMedian,
    firstTextBoxYMedian: firstYMedian,
    shapeCountMedian: median((templateMetrics.slides || []).map((s) => (s.shapes || []).length)),
  },
  ranges: {
    fontSizePt: fontMedian == null ? null : { min: fontMedian - tol(fontMedian), max: fontMedian + tol(fontMedian) },
    firstTextBoxX: firstXMedian == null ? null : { min: firstXMedian - tol(firstXMedian, 0.1, 300000), max: firstXMedian + tol(firstXMedian, 0.1, 300000) },
    firstTextBoxY: firstYMedian == null ? null : { min: firstYMedian - tol(firstYMedian, 0.1, 300000), max: firstYMedian + tol(firstYMedian, 0.1, 300000) },
  },
  expectedFonts: {
    latin: topFonts(latinFreq),
    ea: topFonts(eaFreq),
  },
  hardRules,
  templateSummary: {
    textRuns: allRuns.length,
    shapes: allShapes.length,
    images: (templateMetrics.slides || []).reduce((acc, s) => acc + (s.images || []).length, 0),
  },
};

const outPath = input.output_path || parseArg("--out") || path.resolve(process.cwd(), "outputs", `profile_${Date.now()}.json`);
fs.mkdirSync(path.dirname(outPath), { recursive: true });
fs.writeFileSync(outPath, JSON.stringify(profile, null, 2), "utf-8");

const out = {
  profile_path: outPath,
  slide_count: profile.stats.slideCount,
  font_size_median: profile.stats.fontSizePtMedian,
  hard_rule_count: profile.hardRules.length,
};
if (input.include_profile === true) out.profile = profile;
process.stdout.write(JSON.stringify(out));
