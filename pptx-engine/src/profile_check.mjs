import fs from "fs";
import path from "path";

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

function inRange(v, r) {
  if (v == null || !r) return true;
  return v >= r.min && v <= r.max;
}

function addViolation(list, severity, slideIndex, ruleId, message, evidence) {
  list.push({ severity, slideIndex, ruleId, message, evidence });
}

const input = readInputJSON() || {};

let profile = input.profile || null;
if (!profile) {
  const profilePath = input.profile_path || parseArg("--profile");
  if (!profilePath) {
    console.error("missing profile / --profile");
    process.exit(1);
  }
  profile = JSON.parse(fs.readFileSync(profilePath, "utf-8"));
}

let deckMetrics = input.deck_metrics || null;
if (!deckMetrics) {
  const deckPath = input.deck_metrics_path || parseArg("--deck");
  if (!deckPath) {
    console.error("missing deck_metrics / --deck");
    process.exit(1);
  }
  deckMetrics = JSON.parse(fs.readFileSync(deckPath, "utf-8"));
}

const violations = [];
for (const slide of (deckMetrics.slides || [])) {
  const slideIndex = slide.slideIndex;
  const runs = slide.textRuns || [];
  const imgs = slide.images || [];

  // フォントサイズの逸脱
  for (const r of runs) {
    if (typeof r.fontSizePt === "number" && !inRange(r.fontSizePt, profile.ranges?.fontSizePt)) {
      addViolation(
        violations,
        "B",
        slideIndex,
        "R_FONT_SIZE_RANGE",
        "フォントサイズが型の範囲から外れています",
        {
          fontSizePt: r.fontSizePt,
          expected: profile.ranges?.fontSizePt,
          text: (r.text || "").slice(0, 30),
        },
      );
    }
  }

  // フォント名の逸脱
  for (const r of runs) {
    const latin = r.fontName?.latin;
    const ea = r.fontName?.ea;
    if (latin && profile.expectedFonts?.latin?.length && !profile.expectedFonts.latin.includes(latin)) {
      addViolation(violations, "B", slideIndex, "R_FONT_LATIN", "英字フォントが型と一致していません", {
        actual: latin,
        expectedAnyOf: profile.expectedFonts.latin,
      });
      break;
    }
    if (ea && profile.expectedFonts?.ea?.length && !profile.expectedFonts.ea.includes(ea)) {
      addViolation(violations, "B", slideIndex, "R_FONT_EA", "日本語フォントが型と一致していません", {
        actual: ea,
        expectedAnyOf: profile.expectedFonts.ea,
      });
      break;
    }
  }

  // 先頭テキスト位置の一貫性
  const first = runs[0];
  if (first?.box) {
    if (!inRange(first.box.x, profile.ranges?.firstTextBoxX) || !inRange(first.box.y, profile.ranges?.firstTextBoxY)) {
      addViolation(violations, "B", slideIndex, "R_ANCHOR_POSITION", "先頭テキストの位置が型とずれています", {
        actual: { x: first.box.x, y: first.box.y },
        expectedX: profile.ranges?.firstTextBoxX,
        expectedY: profile.ranges?.firstTextBoxY,
      });
    }
  }

  // 本文行数（簡易）
  for (const r of runs) {
    const lines = (r.text || "").split(/\r?\n/).filter((x) => x.trim()).length;
    if (lines > 2) {
      addViolation(violations, "C", slideIndex, "R_BODY_LINE_COUNT", "本文行数が多すぎます（2行以内推奨）", {
        lines,
        text: (r.text || "").slice(0, 60),
      });
      break;
    }
  }

  // テキストのみ回避（簡易）
  if (imgs.length === 0) {
    addViolation(violations, "C", slideIndex, "R_AVOID_TEXT_ONLY", "図表・画像がなく、テキスト中心です", {
      imageCount: imgs.length,
    });
  }
}

// 重要度順ソート
const order = { A: 0, B: 1, C: 2 };
violations.sort((a, b) => (order[a.severity] - order[b.severity]) || (a.slideIndex - b.slideIndex));

let score = 100;
for (const v of violations) {
  if (v.severity === "A") score -= 20;
  else if (v.severity === "B") score -= 10;
  else score -= 5;
}
if (score < 0) score = 0;

const result = {
  score,
  violations,
  checkedAt: new Date().toISOString(),
};

const outPath = input.output_path || parseArg("--out") || path.resolve(process.cwd(), "outputs", `violations_${Date.now()}.json`);
fs.mkdirSync(path.dirname(outPath), { recursive: true });
fs.writeFileSync(outPath, JSON.stringify(result, null, 2), "utf-8");

const out = {
  violations_path: outPath,
  score: result.score,
  violation_count: result.violations.length,
};
if (input.include_violations === true) out.violations = result.violations;
process.stdout.write(JSON.stringify(out));
