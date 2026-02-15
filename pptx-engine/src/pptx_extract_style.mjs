import fs from "fs";
import path from "path";
import { execSync } from "child_process";

function decodeXml(s) {
  return (s || "")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&amp;/g, "&")
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'");
}

function parseAttrs(tag) {
  const attrs = {};
  const re = /(\w[\w:-]*)="([^"]*)"/g;
  let m;
  while ((m = re.exec(tag))) attrs[m[1]] = m[2];
  return attrs;
}

function findAll(re, s) {
  const out = [];
  let m;
  while ((m = re.exec(s))) out.push(m);
  return out;
}

function emuToPt(v) {
  const n = Number(v || 0);
  if (!Number.isFinite(n)) return null;
  return n / 12700;
}

function parseBoxFromXfrm(block) {
  if (!block) return null;
  const off = block.match(/<a:off\s+[^>]*\/>/);
  const ext = block.match(/<a:ext\s+[^>]*\/>/);
  if (!off || !ext) return null;
  const oa = parseAttrs(off[0]);
  const ea = parseAttrs(ext[0]);
  return {
    x: Number(oa.x || 0),
    y: Number(oa.y || 0),
    w: Number(ea.cx || 0),
    h: Number(ea.cy || 0),
  };
}

function parseColor(block) {
  if (!block) return null;
  const m = block.match(/<a:srgbClr\s+[^>]*val="([0-9A-Fa-f]{6})"/);
  return m ? m[1].toUpperCase() : null;
}

function parseRun(runXml, box, slideIndex) {
  const rpr = runXml.match(/<a:rPr\b[^>]*>[\s\S]*?<\/a:rPr>|<a:rPr\b[^>]*\/>/);
  const rprTag = rpr ? rpr[0].match(/<a:rPr\b[^>]*\/?/)[0] + ">" : "";
  const rprAttrs = parseAttrs(rprTag);
  const latin = rpr ? rpr[0].match(/<a:latin\s+[^>]*typeface="([^"]+)"/) : null;
  const ea = rpr ? rpr[0].match(/<a:ea\s+[^>]*typeface="([^"]+)"/) : null;
  const t = runXml.match(/<a:t>([\s\S]*?)<\/a:t>/);
  return {
    slideIndex,
    text: decodeXml(t ? t[1] : ""),
    fontName: {
      latin: latin ? latin[1] : null,
      ea: ea ? ea[1] : null,
    },
    fontSizePt: rprAttrs.sz ? Number(rprAttrs.sz) / 100 : null,
    bold: rprAttrs.b === "1" || rprAttrs.b === "true",
    color: parseColor(rpr ? rpr[0] : ""),
    box,
  };
}

function parseShape(spXml, slideIndex) {
  const xfrm = spXml.match(/<a:xfrm\b[\s\S]*?<\/a:xfrm>/);
  const box = parseBoxFromXfrm(xfrm ? xfrm[0] : "");

  const spPr = spXml.match(/<p:spPr\b[\s\S]*?<\/p:spPr>/);
  const prst = spPr ? spPr[0].match(/<a:prstGeom\s+[^>]*prst="([^"]+)"/) : null;
  const kind = prst && prst[1] === "line" ? "line" : "rect";
  const ln = spPr ? spPr[0].match(/<a:ln\b[^>]*>[\s\S]*?<\/a:ln>|<a:ln\b[^>]*\/>/) : null;
  const lnAttrs = parseAttrs(ln ? ln[0].match(/<a:ln\b[^>]*\/?/)[0] + ">" : "");

  return {
    kind,
    lineColor: parseColor(ln ? ln[0] : ""),
    fillColor: parseColor(spPr ? spPr[0] : ""),
    lineWidth: lnAttrs.w ? emuToPt(lnAttrs.w) : null,
    box,
    slideIndex,
  };
}

function parseImage(picXml, slideIndex) {
  const xfrm = picXml.match(/<a:xfrm\b[\s\S]*?<\/a:xfrm>/);
  const box = parseBoxFromXfrm(xfrm ? xfrm[0] : "");
  const blip = picXml.match(/<a:blip\b[^>]*r:embed="([^"]+)"/);
  return {
    relId: blip ? blip[1] : null,
    box,
    slideIndex,
  };
}

function shQuote(s) {
  return `'${String(s).replace(/'/g, `'\"'\"'`)}'`;
}

function readZipEntry(pptxPath, entry) {
  try {
    return execSync(`unzip -p ${shQuote(pptxPath)} ${shQuote(entry)}`, { encoding: "utf-8" });
  } catch (e) {
    if (e && typeof e.stdout === "string" && e.stdout.length > 0) return e.stdout;
    throw e;
  }
}

function listZipEntries(pptxPath) {
  let raw = "";
  try {
    raw = execSync(`unzip -Z1 ${shQuote(pptxPath)}`, { encoding: "utf-8" });
  } catch (e) {
    if (e && typeof e.stdout === "string" && e.stdout.length > 0) {
      raw = e.stdout;
    } else {
      throw e;
    }
  }
  return raw.split(/\r?\n/).filter(Boolean);
}

function extractMetrics(pptxPath) {
  const entries = listZipEntries(pptxPath)
    .filter((e) => /^ppt\/slides\/slide\d+\.xml$/.test(e))
    .sort((a, b) => {
      const na = Number(a.match(/slide(\d+)\.xml/)[1]);
      const nb = Number(b.match(/slide(\d+)\.xml/)[1]);
      return na - nb;
    });

  const slides = [];
  for (const entry of entries) {
    const slideIndex = Number(entry.match(/slide(\d+)\.xml/)[1]);
    const xml = readZipEntry(pptxPath, entry);

    const spBlocks = findAll(/<p:sp\b[\s\S]*?<\/p:sp>/g, xml).map((m) => m[0]);
    const picBlocks = findAll(/<p:pic\b[\s\S]*?<\/p:pic>/g, xml).map((m) => m[0]);

    const textRuns = [];
    const shapes = [];
    const images = [];

    for (const sp of spBlocks) {
      shapes.push(parseShape(sp, slideIndex));
      const xfrm = sp.match(/<a:xfrm\b[\s\S]*?<\/a:xfrm>/);
      const box = parseBoxFromXfrm(xfrm ? xfrm[0] : "");
      const runs = findAll(/<a:r\b[\s\S]*?<\/a:r>/g, sp).map((m) => m[0]);
      for (const r of runs) textRuns.push(parseRun(r, box, slideIndex));
    }

    for (const pic of picBlocks) images.push(parseImage(pic, slideIndex));

    slides.push({
      slideIndex,
      textRuns,
      shapes,
      images,
    });
  }

  return {
    sourcePptx: path.resolve(pptxPath),
    slideCount: slides.length,
    slides,
  };
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
const pptxPath = input.pptx_path || parseArg("--pptx");
if (!pptxPath) {
  console.error("missing pptx_path / --pptx");
  process.exit(1);
}

const metrics = extractMetrics(pptxPath);
const outPath = input.output_path || parseArg("--out") || path.resolve(process.cwd(), "outputs", `metrics_${Date.now()}.json`);
fs.mkdirSync(path.dirname(outPath), { recursive: true });
fs.writeFileSync(outPath, JSON.stringify(metrics, null, 2), "utf-8");

const out = {
  metrics_path: outPath,
  slide_count: metrics.slideCount,
};
if (input.include_metrics === true) out.metrics = metrics;
process.stdout.write(JSON.stringify(out));
