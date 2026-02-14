import fs from "fs";
import path from "path";
import PptxGenJS from "pptxgenjs";

function readStdin() {
  /*標準入力からデータを読み取る関数*/
  return fs.readFileSync(0, "utf-8");
}

function ensureDir(p) {
  /*指定されたパスにディレクトリが存在しない場合は作成する関数*/
  fs.mkdirSync(p, { recursive: true });
}

function addBullets(slide, title, bullets) {
  /*スライドにタイトルと箇条書きを追加する関数*/
  slide.addText(title, { x: 0.6, y: 0.4, w: 12.1, h: 0.6, fontSize: 28, bold: true });
  const text = (bullets || []).map(b => `• ${b}`).join("\n");
  slide.addText(text, { x: 0.9, y: 1.3, w: 12.0, h: 5.2, fontSize: 20 });
}

// 入力データの読み込みと出力ディレクトリの準備
const input = JSON.parse(readStdin());
const outDir = path.resolve(process.cwd(), "..", "output");
ensureDir(outDir);

const pptx = new PptxGenJS();
pptx.layout = "LAYOUT_WIDE"; // 16:9 レイアウト

// タイトルスライド
{
  const s = pptx.addSlide();
  s.addText(input?.title || "SlideAgent Deck", { x: 0.8, y: 1.6, w: 12.0, h: 1.0, fontSize: 40, bold: true });
  if (input?.subtitle) {
    s.addText(input.subtitle, { x: 0.9, y: 2.7, w: 12.0, h: 0.8, fontSize: 22 });
  }
}

// 本文スライド
for (const slideSpec of (input.slides || [])) {
  const s = pptx.addSlide();
  addBullets(s, slideSpec.title || "Untitled", slideSpec.bullets || []);
  // 画像を挿入（任意）
  if (slideSpec.imagePath) {
    s.addImage({ path: slideSpec.imagePath, x: 7.0, y: 1.6, w: 5.5, h: 3.5 });
  }
}

const filename = input?.outputName || `deck_${Date.now()}.pptx`;
const outPath = path.join(outDir, filename);
await pptx.writeFile({ fileName: outPath });

process.stdout.write(JSON.stringify({ pptx_path: outPath }));
