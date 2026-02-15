import fs from "fs";
import path from "path";

function parseArg(flag) {
  const i = process.argv.indexOf(flag);
  if (i >= 0 && i + 1 < process.argv.length) return process.argv[i + 1];
  return null;
}

const profilePath = parseArg("--profile") || "outputs/profile.json";
const violationsPath = parseArg("--violations") || "outputs/violations.json";
const outPath = parseArg("--out") || path.resolve(process.cwd(), "outputs", `tips_${Date.now()}.json`);

const apiKey = process.env.GEMINI_API_KEY || "";
if (!apiKey) {
  console.error("GEMINI_API_KEY is empty");
  process.exit(1);
}
const model = process.env.GEMINI_MODEL || "gemini-2.5-flash";

const templateProfile = JSON.parse(fs.readFileSync(profilePath, "utf-8"));
const violations = JSON.parse(fs.readFileSync(violationsPath, "utf-8"));

const prompt = `あなたは「スライド設計コーチ」です。ユーザーはハッカソン/研究発表のスライドを改善したいが、単なる修正案ではなく“原理（コツ）”を学びたい。

入力:
- template_profile: 型のルールと統計
- violations: 型からのズレ一覧（evidence付き）

出力（日本語）:
1) 総合スコア（100点満点）と一言講評（15〜25字）
2) 重要度A（最大3件）：各項目を「なぜ重要か（原理）」→「今の状態（evidence）」→「直し方（具体）」の順で説明
3) 重要度B（最大3件）：同様に短く
4) “型の要点まとめ”を5行で（暗記できる形）
5) 次に見るべきチェックリスト（7項目）

制約:
- 断定しすぎず、evidenceに基づく
- “直し方”はユーザーが手で直せる粒度（位置/フォント/行数など）
- 箇条書き中心。必要な専門用語は一言で補足。


template_profile:
${JSON.stringify(templateProfile, null, 2)}

violations:
${JSON.stringify(violations, null, 2)}
`;

const url = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent`;
const resp = await fetch(url, {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "x-goog-api-key": apiKey,
  },
  body: JSON.stringify({
    contents: [{ parts: [{ text: prompt }] }],
  }),
});

if (!resp.ok) {
  const txt = await resp.text();
  console.error(`gemini api error: status=${resp.status} body=${txt}`);
  process.exit(1);
}

const data = await resp.json();
const text = data?.candidates?.[0]?.content?.parts?.[0]?.text || "";
if (!text) {
  console.error("empty gemini response");
  process.exit(1);
}

const out = {
  generatedAt: new Date().toISOString(),
  profilePath,
  violationsPath,
  tips: text,
};
fs.mkdirSync(path.dirname(outPath), { recursive: true });
fs.writeFileSync(outPath, JSON.stringify(out, null, 2), "utf-8");
process.stdout.write(JSON.stringify({ tips_path: outPath }));
