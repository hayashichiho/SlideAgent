import fs from "fs";
import path from "path";

function readStdin() {
  /*標準入力からデータを読み取る関数*/
  return fs.readFileSync(0, "utf-8");
}
function ensureDir(p) {
  /*指定されたパスにディレクトリが存在しない場合は作成する関数*/
  fs.mkdirSync(p, { recursive: true });
}

// 入力データの読み込みと出力ディレクトリの準備
const input = JSON.parse(readStdin());
const steps = (input.steps || []).slice(0, 6);
const w = 1200, h = 400;
const boxW = 180, boxH = 90, gap = 20;
const startX = 40, y = 150;

// SVGのヘッダーと背景を作成
let svg = [];
svg.push(`<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}">`);
svg.push(`<rect width="100%" height="100%" fill="white"/>`);

// 各ステップのボックスと矢印を描画
steps.forEach((t, i) => {
  const x = startX + i * (boxW + gap);
  svg.push(`<rect x="${x}" y="${y}" width="${boxW}" height="${boxH}" rx="18" ry="18" fill="#f5f5f5" stroke="#222" stroke-width="2"/>`);
  svg.push(`<text x="${x + boxW / 2}" y="${y + boxH / 2}" text-anchor="middle" dominant-baseline="middle" font-family="Arial" font-size="22">${t}</text>`);
  if (i < steps.length - 1) {
    const ax = x + boxW;
    const bx = x + boxW + gap;
    const my = y + boxH / 2;
    svg.push(`<line x1="${ax}" y1="${my}" x2="${bx}" y2="${my}" stroke="#222" stroke-width="3"/>`);
    svg.push(`<polygon points="${bx},${my} ${bx - 16},${my - 10} ${bx - 16},${my + 10}" fill="#222"/>`);
  }
});

svg.push(`</svg>`);
const outDir = path.resolve(process.cwd(), "..", "output");
ensureDir(outDir);
const outPath = path.join(outDir, input.outputName || `diagram_${Date.now()}.svg`);
// 生成したSVGを保存
fs.writeFileSync(outPath, svg.join("\n"), "utf-8");

process.stdout.write(JSON.stringify({ image_path: outPath }));
