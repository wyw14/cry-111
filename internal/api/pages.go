package api

import (
	"fmt"
	"net/http"
)

func (s *Server) routesPage(w http.ResponseWriter, _ *http.Request) {
	servePage(w, "进路控制", "routes", "/api/routes", "申请进路", "铁路进路锁闭与运行阶段")
}

func (s *Server) pointsPage(w http.ResponseWriter, _ *http.Request) {
	servePage(w, "道岔监控", "points", "/api/points", "转换道岔", "位置、密贴与锁闭证明")
}

func (s *Server) signalsPage(w http.ResponseWriter, _ *http.Request) {
	servePage(w, "信号机", "signals", "/api/signals", "检查回路", "命令灯位、选择反馈与灯丝证明")
}

func (s *Server) incidentsPage(w http.ResponseWriter, _ *http.Request) {
	servePage(w, "安全事件", "incidents", "/api/incidents", "确认事件", "联锁告警与现场处置状态")
}

func servePage(w http.ResponseWriter, title, view, endpoint, action, subtitle string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, pageDocument, title, view, endpoint, title, subtitle, action)
}

func (s *Server) stylesheet(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(pageStyles))
}

func (s *Server) javascript(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write([]byte(pageScript))
}

const pageDocument = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>SignalRoute · %s</title>
<link rel="stylesheet" href="/assets/app.css">
</head>
<body data-view="%s" data-endpoint="%s">
<header class="topbar">
<a class="brand" href="/routes"><span class="signal-mark"><i></i><i></i><i></i></span><strong>SignalRoute</strong></a>
<nav><a href="/routes">进路</a><a href="/points">道岔</a><a href="/signals">信号</a><a href="/incidents">事件</a></nav>
<div class="health"><span></span>联锁在线</div>
</header>
<main>
<section class="heading"><div><h1>%s</h1><p>%s</p></div><button id="primary-action" type="button">%s</button></section>
<section class="rail-map" aria-label="站场状态"><div class="track-line"><b>1DG</b><span></span><b>5DG</b><span></span><b>11DG</b></div><div class="route-state">R14 <strong>PROVING</strong></div></section>
<section class="summary"><div><span>活动进路</span><strong id="metric-primary">0</strong></div><div><span>安全资源</span><strong id="metric-secondary">0</strong></div><div><span>最后刷新</span><strong id="last-refresh">--:--:--</strong></div></section>
<section class="data-panel"><div class="panel-title"><h2>实时状态</h2><button id="refresh" type="button" title="刷新">↻</button></div><div id="status" class="status">正在读取现场状态</div><div id="data-grid" class="data-grid"></div></section>
</main>
<script src="/assets/app.js"></script>
</body>
</html>`

const pageStyles = `:root{color-scheme:dark;font-family:Inter,Segoe UI,Arial,sans-serif;background:#090d0c;color:#e9efec}*{box-sizing:border-box}body{margin:0;min-height:100vh;background:#090d0c}.topbar{height:64px;display:flex;align-items:center;border-bottom:1px solid #26302c;padding:0 28px;background:#0e1412;position:sticky;top:0;z-index:3}.brand{display:flex;align-items:center;gap:12px;color:#fff;text-decoration:none;min-width:210px}.brand strong{font-size:17px}.signal-mark{display:flex;gap:3px;padding:4px 7px;border:1px solid #45524d;border-radius:4px}.signal-mark i{width:7px;height:7px;border-radius:50%;background:#47534f}.signal-mark i:nth-child(2){background:#efb848}.signal-mark i:nth-child(3){background:#44cf7a}nav{display:flex;align-self:stretch}nav a{display:flex;align-items:center;padding:0 18px;color:#9caaa4;text-decoration:none;border-bottom:2px solid transparent}nav a:hover{color:#fff;border-bottom-color:#efb848}.health{margin-left:auto;color:#aab6b1;font-size:13px;display:flex;align-items:center;gap:8px}.health span{width:8px;height:8px;border-radius:50%;background:#44cf7a;box-shadow:0 0 0 3px #143421}main{width:min(1180px,calc(100% - 40px));margin:0 auto;padding:32px 0 56px}.heading{display:flex;align-items:end;justify-content:space-between;margin-bottom:26px}.heading h1{font-size:28px;margin:0 0 7px;letter-spacing:0}.heading p{margin:0;color:#8f9b96}.heading button,.panel-title button{border:1px solid #3c4944;background:#17211d;color:#eef4f1;border-radius:5px;height:38px;padding:0 15px;font-weight:600}.heading button:hover,.panel-title button:hover{border-color:#efb848}.rail-map{border:1px solid #2b3833;background:#101714;border-radius:6px;padding:24px;margin-bottom:18px;display:grid;grid-template-columns:1fr auto;align-items:center}.track-line{display:flex;align-items:center;color:#bac5c1;font-size:12px}.track-line span{height:3px;background:#44cf7a;flex:1;margin:0 8px;position:relative}.track-line span:after{content:"";position:absolute;right:30%;top:-6px;width:15px;height:15px;border:2px solid #efb848;border-radius:50%;background:#101714}.route-state{padding:9px 12px;border-left:1px solid #34413c;color:#8f9b96}.route-state strong{color:#efb848;margin-left:10px}.summary{display:grid;grid-template-columns:repeat(3,1fr);border:1px solid #28342f;border-radius:6px;margin-bottom:18px;background:#0f1613}.summary div{padding:18px 20px;border-right:1px solid #28342f}.summary div:last-child{border:0}.summary span{display:block;color:#84918c;font-size:12px;margin-bottom:7px}.summary strong{font-size:20px}.data-panel{border-top:1px solid #2b3833;padding-top:18px}.panel-title{display:flex;justify-content:space-between;align-items:center}.panel-title h2{font-size:16px;margin:0}.panel-title button{width:38px;padding:0;font-size:20px}.status{color:#84918c;font-size:13px;margin:14px 0}.data-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1px;background:#28342f;border:1px solid #28342f}.data-row{background:#0f1613;padding:16px;min-height:98px}.data-row .label{font-weight:700;margin-bottom:10px}.data-row dl{display:grid;grid-template-columns:repeat(2,1fr);gap:6px;margin:0;font-size:12px}.data-row dt{color:#7f8c87}.data-row dd{margin:0;text-align:right;overflow-wrap:anywhere;color:#dce5e1}.tag{display:inline-block;border:1px solid #44524c;border-radius:3px;padding:2px 6px;font-size:11px;color:#efb848}@media(max-width:720px){.topbar{padding:0 14px}.brand{min-width:0}.brand strong{display:none}nav a{padding:0 9px;font-size:13px}.health{display:none}main{width:min(100% - 24px,1180px);padding-top:22px}.heading{align-items:start;gap:18px}.heading h1{font-size:23px}.rail-map{grid-template-columns:1fr}.route-state{border-left:0;border-top:1px solid #34413c;margin-top:18px;padding-left:0}.summary{grid-template-columns:1fr}.summary div{border-right:0;border-bottom:1px solid #28342f}.data-grid{grid-template-columns:1fr}}`

const pageScript = `const body=document.body;const endpoint=body.dataset.endpoint;const grid=document.getElementById('data-grid');const statusNode=document.getElementById('status');const metricPrimary=document.getElementById('metric-primary');const metricSecondary=document.getElementById('metric-secondary');const lastRefresh=document.getElementById('last-refresh');function scalar(value){if(value===null||value===undefined)return '—';if(typeof value==='boolean')return value?'是':'否';if(Array.isArray(value))return value.join(', ')||'—';if(typeof value==='object')return JSON.stringify(value);return String(value)}function rows(payload){const key=Object.keys(payload).find(k=>Array.isArray(payload[k]));return key?payload[key]:[]}function renderItem(item,index){const entries=Object.entries(item).slice(0,6);const title=item.name||item.id||item.route_id||('记录 '+(index+1));return '<article class="data-row"><div class="label">'+scalar(title)+' <span class="tag">'+scalar(item.phase||item.state||item.severity||'ACTIVE')+'</span></div><dl>'+entries.map(([key,value])=>'<dt>'+key.replaceAll('_',' ')+'</dt><dd>'+scalar(value)+'</dd>').join('')+'</dl></article>'}async function refresh(){statusNode.textContent='正在同步联锁状态';try{const response=await fetch(endpoint,{headers:{Accept:'application/json'}});if(!response.ok)throw new Error('HTTP '+response.status);const payload=await response.json();const items=rows(payload);grid.innerHTML=items.map(renderItem).join('')||'<article class="data-row"><div class="label">当前没有活动记录</div></article>';metricPrimary.textContent=String(items.length);const secondary=Object.values(payload).find(v=>Array.isArray(v)&&v!==items);metricSecondary.textContent=String(secondary?secondary.length:items.filter(v=>v.active!==false).length);lastRefresh.textContent=new Date().toLocaleTimeString('zh-CN',{hour12:false});statusNode.textContent='现场状态已同步'}catch(error){statusNode.textContent='读取失败 '+error.message;grid.innerHTML=''}}document.getElementById('refresh').addEventListener('click',refresh);document.getElementById('primary-action').addEventListener('click',refresh);refresh();setInterval(refresh,8000);`
