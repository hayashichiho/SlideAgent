package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var pageTmpl = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="ja">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>SlideAgent Web MVP</title>
  <style>
    :root { --bg: #f4f6f8; --card: #fff; --ink: #111; --muted:#666; --accent:#0b5cff; }
    body { margin: 0; font-family: "Noto Sans JP", sans-serif; background: var(--bg); color: var(--ink); }
    .wrap { max-width: 920px; margin: 24px auto; padding: 0 16px; }
    .card { background: var(--card); border-radius: 12px; padding: 16px; box-shadow: 0 6px 18px rgba(0,0,0,.06); }
    h1 { margin: 0 0 12px; font-size: 22px; }
    textarea { width: 100%; min-height: 120px; resize: vertical; font-size: 14px; padding: 10px; }
    input, button { font-size: 14px; padding: 8px 10px; }
    .row { display: flex; gap: 8px; align-items: center; margin-top: 10px; }
    button { background: var(--accent); color: #fff; border: none; border-radius: 8px; cursor: pointer; }
    button:disabled { opacity: .6; cursor: not-allowed; }
    .mt { margin-top: 16px; }
    pre { background: #0f1720; color: #e5e7eb; padding: 12px; border-radius: 8px; overflow:auto; }
    .muted { color: var(--muted); font-size: 13px; }
    ul { margin-top: 8px; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <h1>SlideAgent Web MVP</h1>
      <p class="muted">入力して「ジョブ作成」を押すと、状態を自動更新します。</p>
      <textarea id="pitch" placeholder="ピッチ文を入力"></textarea>
      <div class="row">
        <label>maxAttempts</label>
        <input id="maxAttempts" type="number" min="1" value="2" style="width:90px" />
        <button id="submitBtn">ジョブ作成</button>
      </div>
    </div>

    <div class="card mt">
      <h1>ジョブ状態</h1>
      <div id="jobMeta" class="muted">未作成</div>
      <pre id="jobJson">{}</pre>
      <h1>成果物</h1>
      <ul id="artifacts"></ul>
    </div>
  </div>

<script>
const pitchEl = document.getElementById('pitch');
const maxAttemptsEl = document.getElementById('maxAttempts');
const submitBtn = document.getElementById('submitBtn');
const jobMeta = document.getElementById('jobMeta');
const jobJson = document.getElementById('jobJson');
const artifactsEl = document.getElementById('artifacts');
let pollTimer = null;

function setBusy(b){ submitBtn.disabled = b; }
function renderArtifacts(items){
  artifactsEl.innerHTML = '';
  if(!items || !items.length){
    const li = document.createElement('li');
    li.textContent = 'なし';
    artifactsEl.appendChild(li);
    return;
  }
  for (const a of items){
    const li = document.createElement('li');
    const code = document.createElement('code');
    code.textContent = a.type + ': ' + a.path;
    li.appendChild(code);
    artifactsEl.appendChild(li);
  }
}

async function fetchJSON(path, init){
  const res = await fetch(path, init);
  const txt = await res.text();
  let data = {};
  try { data = txt ? JSON.parse(txt) : {}; } catch {}
  if(!res.ok){
    throw new Error(data.error || (String(res.status) + ' ' + res.statusText));
  }
  return data;
}

async function pollJob(id){
  try{
    const job = await fetchJSON('/api/jobs/' + id);
    jobMeta.textContent = 'jobId=' + job.id + ' status=' + job.status + ' attempts=' + job.attempts + '/' + job.maxAttempts;
    jobJson.textContent = JSON.stringify(job, null, 2);

    if(job.status === 'succeeded'){
      const arts = await fetchJSON('/api/jobs/' + id + '/artifacts');
      renderArtifacts(arts.artifacts);
      clearInterval(pollTimer);
      pollTimer = null;
    }
    if(job.status === 'failed'){
      renderArtifacts([]);
      clearInterval(pollTimer);
      pollTimer = null;
    }
  } catch (e){
    jobMeta.textContent = 'poll error: ' + e.message;
  }
}

submitBtn.addEventListener('click', async () => {
  const pitchText = pitchEl.value.trim();
  if(!pitchText){ alert('pitchText を入力してください'); return; }
  const maxAttempts = Number(maxAttemptsEl.value || '2');
  setBusy(true);
  try{
    const created = await fetchJSON('/api/jobs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pitchText, maxAttempts })
    });
    const id = created.id;
    jobMeta.textContent = 'jobId=' + id + ' status=' + created.status;
    jobJson.textContent = JSON.stringify(created, null, 2);
    renderArtifacts([]);
    if(pollTimer){ clearInterval(pollTimer); }
    pollTimer = setInterval(() => pollJob(id), 2500);
    await pollJob(id);
  } catch (e){
    jobMeta.textContent = 'submit error: ' + e.message;
  } finally {
    setBusy(false);
  }
});
</script>
</body>
</html>`))

func apiBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("API_BASE_URL")); v != "" {
		return v
	}
	return "http://127.0.0.1:8080"
}

func frontendAddr() string {
	if v := strings.TrimSpace(os.Getenv("FRONTEND_ADDR")); v != "" {
		return v
	}
	return ":3000"
}

func main() {
	apiURL, err := url.Parse(apiBaseURL())
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(apiURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		_ = pageTmpl.Execute(w, nil)
	})
	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
		proxy.ServeHTTP(w, r)
	}))

	addr := frontendAddr()
	fmt.Println("frontend listening on", addr, "proxying api to", apiURL.String())
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}
