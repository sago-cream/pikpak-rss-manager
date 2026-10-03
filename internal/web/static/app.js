'use strict';
const $ = (selector) => document.querySelector(selector);
const escapeHTML = (value) => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let csrf = '', toastTimer;
async function api(path, method = 'GET', body) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 110000);
  try {
    const response = await fetch(path, {method, credentials: 'same-origin', signal: controller.signal,
      headers: {'Content-Type':'application/json', 'X-CSRF-Token':csrf}, body: body === undefined ? undefined : JSON.stringify(body)});
    const result = await response.json();
    if (!response.ok) {
      if (response.status === 401 && path !== '/api/login' && !$('#login-form')) location.reload();
      throw new Error(result.error || '操作未完成，請稍後再試');
    }
    return result;
  } catch (error) { if (error.name === 'AbortError') throw new Error('操作逾時，請檢查任務紀錄後再試'); throw error; }
  finally { clearTimeout(timer); }
}
function toast(message, error = false) {
  const element = $('#toast'); if (!element) return;
  clearTimeout(toastTimer); element.textContent = message; element.className = `toast ${error ? 'is-error' : ''}`;
  toastTimer = setTimeout(() => element.classList.add('hidden'), 4500);
}
function date(timestamp) { return timestamp ? new Date(timestamp * 1000).toLocaleString('zh-TW', {month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hour12:false}) : '尚未檢查'; }
const labels = {queued:'等待處理',submitting:'正在提交',submission_unknown:'提交待核對',downloading:'雲端下載中',organizing:'正在整理',complete:'已完成',needs_review:'待處理',failed:'下載失敗',paused_auth:'授權暫停',paused_quota:'配額暫停',paused_account:'帳號暫停'};
const runningStates = ['queued','submitting','downloading','organizing'];
const reviewStates = ['needs_review','submission_unknown','failed','paused_auth','paused_quota','paused_account'];
function badge(state) { return `<span class="badge ${state === 'complete' ? 'green' : reviewStates.includes(state) ? 'orange' : 'purple'}">${escapeHTML(labels[state] || state)}</span>`; }
const views = {
  overview:['你的訂閱，一目了然。','追蹤最新發布，讓雲端自動完成下載與整理。','總覽'],
  subscriptions:['為喜歡的作品，留一個位置。','每筆訂閱，都有自己的目錄與命名規則。','訂閱管理'],
  jobs:['每個任務，都有跡可循。','從離線下載到逐檔整理，查看每一步的結果。','離線任務'],
  events:['所有動作，清楚記錄。','檢查來源、觸發任務、整理結果，都在這裡。','執行日誌'],
  settings:['連接你的 PikPak。','以官方授權安全操作雲端，無需提供帳號密碼。','授權設定']
};
let data = {subscriptions:[], jobs:[], events:[], status:{}}, currentView = 'overview', jobFilter = 'all';
function showView(name) {
  if (!views[name]) return;
  currentView = name;
  for (const element of document.querySelectorAll('.view')) element.classList.toggle('hidden', element.id !== `view-${name}`);
  for (const element of document.querySelectorAll('.nav-item')) element.classList.toggle('active', element.dataset.view === name);
  [$('#page-title').textContent,$('#page-description').textContent,$('#breadcrumb-title').textContent] = views[name];
  for (const button of document.querySelectorAll('.heading-actions .new-subscription')) button.classList.toggle('hidden', name === 'settings' || name === 'events');
  history.replaceState(null,'',`#${name}`);
}
function empty(title, description, button = '') { return `<div class="empty-state"><span class="empty-symbol">⌁</span><h3>${escapeHTML(title)}</h3><p>${escapeHTML(description)}</p>${button}</div>`; }
function subscriptionCards(items) {
  if (!items.length) return empty('從第一筆訂閱開始','加入 RSS，設定作品的目錄與檔名規則。','<button class="button primary new-subscription">＋ 新增訂閱</button>');
  return items.map((s,i) => `<article class="subscription-card"><div class="card-top"><span class="subscription-icon color-${i%4}">${escapeHTML(s.name.slice(0,1))}</span><span class="badge ${s.enabled ? 'green' : ''}">${s.enabled ? '追蹤中' : '已停用'}</span></div><h3>${escapeHTML(s.name)}</h3><div class="card-path"><span>↳</span>${escapeHTML(s.destination || '根目錄')}</div><div class="card-rule"><span>${s.rename_enabled === false ? '保留原始檔名' : s.rename_mode === 'replace' ? 'Regex 尋找／替換' : '命名範本'}</span><code>${s.rename_enabled === false ? '只移動，不重命名' : escapeHTML(s.rename_mode === 'replace' ? `${s.regex} → ${s.replacement || '（移除匹配部分）'}` : s.template)}</code></div>${s.last_error ? `<p class="error compact">${escapeHTML(s.last_error)}</p>` : ''}<div class="card-meta"><span>每 ${s.interval_minutes} 分鐘</span><span>${date(s.last_checked)}</span></div><div class="card-actions"><button class="text-button" data-edit="${s.id}">編輯訂閱</button><div><button class="text-button" data-check="${s.id}" title="檢查新的發布">檢查</button><button class="text-button" data-backfill="${s.id}" title="處理 RSS 中仍可取得的現有項目">補抓</button><button class="icon-button danger" data-delete="${s.id}" aria-label="刪除 ${escapeHTML(s.name)}">×</button></div></div></article>`).join('');
}
function jobsTable(items) {
  if (!items.length) return empty('還沒有離線任務','首次訂閱只建立基準；新發布或補抓項目會出現在這裡。');
  return `<div class="table-scroll"><table><thead><tr><th>發布與作品</th><th>狀態</th><th>時間</th><th></th></tr></thead><tbody>${items.map(j=>`<tr><td class="job-name"><strong>${escapeHTML(j.rule.title)}</strong><span title="${escapeHTML(j.title)}">${escapeHTML(j.title)}</span>${j.error ? `<small class="error">${escapeHTML(j.error)}</small>` : ''}</td><td>${badge(j.state)}${j.state==='downloading' ? `<div class="progress-caption">${Math.round(j.progress)}%</div>` : ''}</td><td class="muted nowrap">${date(j.created_at)}</td><td><button class="text-button" data-job="${j.id}">查看 →</button></td></tr>`).join('')}</tbody></table></div>`;
}
function render() {
  $('#stat-subscriptions').textContent=data.subscriptions.filter(s=>s.enabled).length;
  $('#stat-running').textContent=data.jobs.filter(j=>runningStates.includes(j.state)).length;
  $('#stat-complete').textContent=data.jobs.filter(j=>j.state==='complete').length;
  $('#stat-review').textContent=data.jobs.filter(j=>reviewStates.includes(j.state)).length;
  $('#overview-subscriptions').innerHTML=subscriptionCards(data.subscriptions.slice(0,3));
  const search=($('#subscription-search').value||'').toLowerCase();
  $('#subscriptions-list').innerHTML=subscriptionCards(data.subscriptions.filter(s=>s.name.toLowerCase().includes(search)));
  $('#subscription-count').textContent=`共 ${data.subscriptions.length} 筆訂閱`;
  $('#overview-jobs').innerHTML=jobsTable(data.jobs.slice(0,5));
  $('#jobs-list').innerHTML=jobsTable(data.jobs.filter(j=>jobFilter==='all'||(jobFilter==='running'&&runningStates.includes(j.state))||(jobFilter==='review'&&reviewStates.includes(j.state))||(jobFilter==='complete'&&j.state==='complete')));
  $('#events-list').innerHTML=data.events.length ? data.events.map(e=>`<div class="event-row"><span class="event-dot ${escapeHTML(e.level)}"></span><div><p>${escapeHTML(e.message)}</p><small class="muted">${e.job_id ? `任務 ${escapeHTML(e.job_id.slice(0,8))}` : e.subscription_id ? `訂閱 #${e.subscription_id}` : '系統'}</small></div><time>${date(e.created_at)}</time></div>`).join('') : empty('一切從這裡開始','執行檢查或離線任務後，這裡會顯示紀錄。');
  const status=data.status;
  const connected=status.connected&&!status.paused;
  $('#sidebar-connection').innerHTML=`<span class="dot ${connected?'green':'amber'}"></span>${connected?'PikPak 已連線':status.paused?'PikPak 已暫停':'尚未綁定 PikPak'}`;
  $('#settings-badge').className=`badge ${connected?'green':'orange'}`;
  $('#settings-badge').textContent=connected?'連線正常':status.paused?'已暫停':'未綁定';
  $('#connection-banner').classList.toggle('hidden',connected);
  $('#connection-banner').textContent=status.message||(status.connected ? '請重新檢查 PikPak 連線。' : '先到「授權設定」綁定 PikPak，再開始自動追蹤。');
  $('#account-details').innerHTML=status.connected?`<div class="account-card"><span class="account-avatar">P</span><div><strong>${escapeHTML(status.name||'PikPak 帳號')}</strong><span>${formatBytes(status.storage_used)} / ${formatBytes(status.storage_total)} 雲端空間</span></div></div>`:'';
  $('#token').disabled=!!status.external; $('#save-token').disabled=!!status.external;
  $('#token-source').textContent=status.external?'目前由環境變數／私密檔案提供 PAT。更新該設定後重新啟動服務。':'PAT 只用於官方 PikPak 端點；儲存後加密，不會回傳至瀏覽器。';
}
function formatBytes(value) { let n=Number(value||0); if (!Number.isFinite(n)) return '—'; const units=['B','KiB','MiB','GiB','TiB']; let i=0; while(n>=1024&&i<4){n/=1024;i++;} return `${n.toFixed(i?1:0)} ${units[i]}`; }
async function load(silent=false) { try { const [subscriptions,jobs,events,status]=await Promise.all(['/api/subscriptions','/api/jobs','/api/events','/api/settings/pikpak'].map(p=>api(p))); data={subscriptions,jobs,events,status}; render(); } catch(e){if(!silent)toast(e.message,true);} }
const defaultRegex='(?i)(?:S(?P<season>[0-9]{1,2})E(?P<ep>[0-9]{1,3})|第\\s*(?P<ep>[0-9]{1,3})\\s*[話话集]|(?:^|[\\s\\[\\]_-])(?P<ep>[0-9]{1,3})(?:v[0-9]+)?(?:$|[\\s\\[\\]_.-]))';
function openSubscription(id) {
  const sub=data.subscriptions.find(s=>s.id===Number(id)); $('#subscription-form').reset();
  $('#sub-id').value=sub?.id||''; $('#dialog-title').textContent=sub?'編輯訂閱':'新增訂閱';
  $('#sub-name').value=sub?.name||''; $('#sub-url').value=sub?.rss_url||''; $('#sub-destination').value=sub?.destination||'';
  $('#sub-destination-id').value=sub?.destination_id||''; $('#sub-destination-account-ref').value=sub?.destination_account_ref||'';
  $('#sub-interval').value=sub?.interval_minutes||10; $('#sub-season').value=sub?.season||1; $('#sub-enabled').checked=sub?.enabled??true;
  $('#sub-rename-enabled').checked=sub ? sub.rename_enabled??true : false;
  $('#sub-rename-mode').value=sub?.rename_mode||(sub?'template':'replace');
  $('#rename-mode-label').classList.toggle('hidden',!sub||sub.rename_mode==='replace');
  $('#sub-template').value=sub?.template||'{title} - S{season:02}E{ep:02}.{ext}';
  $('#sub-regex').value=sub?.regex??'^\\[[^\\]]+\\]\\s*'; $('#sub-replacement').value=sub?.replacement||'';
  closeFolderBrowser(); updateRenameOptions(); $('#browse-folders').disabled=!data.status.connected||!!data.status.paused;
  $('#destination-help').textContent=data.status.connected?'從 PikPak 選取資料夾，也可手動輸入路徑。':'綁定 PikPak PAT 後即可列出或建立資料夾；也可先手動輸入路徑。';
  $('#subscription-dialog').showModal();
}
function formRule(){return {title:$('#sub-name').value.trim(),season:Number($('#sub-season').value),regex:$('#sub-regex').value,template:$('#sub-template').value,rename_enabled:$('#sub-rename-enabled').checked,mode:$('#sub-rename-mode').value,replacement:$('#sub-replacement').value};}
function updateRenameOptions(){
  const enabled=$('#sub-rename-enabled').checked, legacy=$('#sub-rename-mode').value==='template';
  $('#rename-options').classList.toggle('hidden',!enabled); $('#legacy-options').classList.toggle('hidden',!legacy); $('#replacement-options').classList.toggle('hidden',legacy); $('#preview-title-label').classList.toggle('hidden',!legacy);
  for(const id of ['sub-regex','sub-replacement','sub-template','sub-season','sub-rename-mode'])$('#'+id).disabled=!enabled;
  $('#sub-template').required=enabled&&legacy; $('#sub-season').required=enabled&&legacy; $('#sub-template').disabled=!enabled||!legacy; $('#sub-season').disabled=!enabled||!legacy; $('#sub-regex').required=enabled&&!legacy;
  $('#regex-label').textContent=legacy?'提取 Regex（既有規則）':'尋找 Regex';
  $('#preview-result').className='preview-result'; $('#preview-result').textContent='輸入原始檔名，預覽替換結果。';
}

let folderState=null, folderRequest=0;
function closeFolderBrowser(){folderRequest++;folderState=null;$('#folder-browser').classList.add('hidden');$('#folder-list').innerHTML='';$('#new-folder-name').value='';}
function renderFolders(){
  if(!folderState)return;
  $('#folder-breadcrumbs').innerHTML=folderState.breadcrumbs.map(d=>`<button type="button" class="text-button" data-folder="${escapeHTML(d.id)}">${escapeHTML(d.name)}</button>`).join('<span>/</span>');
  const term=$('#folder-search').value.toLowerCase(), folders=folderState.folders.filter(f=>f.name.toLowerCase().includes(term));
  $('#folder-list').innerHTML=folders.length?folders.map(f=>`<button type="button" class="folder-row" data-folder="${escapeHTML(f.id)}"><span class="folder-symbol">▱</span><span>${escapeHTML(f.name)}</span><span class="folder-arrow">›</span></button>`).join(''):'<p class="field-help">目前沒有符合的資料夾。</p>';
  $('#folder-more').classList.toggle('hidden',!folderState.next_token); $('#folder-current').textContent=`目前位置：${folderState.current.path||'根目錄'}`;
  $('#folder-select').textContent=folderState.current.id?'使用此資料夾':'使用根目錄'; $('#folder-select').disabled=false; $('#folder-create').disabled=false;
}
async function loadFolders(parent='',token='',append=false){
  const sequence=++folderRequest; $('#folder-browser').classList.remove('hidden'); $('#folder-status').className='field-help'; $('#folder-status').textContent='正在讀取 PikPak 資料夾…';
  $('#folder-select').disabled=true; $('#folder-create').disabled=true; $('#folder-more').disabled=true;
  if(!append){folderState=null;$('#folder-list').innerHTML='';$('#folder-breadcrumbs').innerHTML='';$('#folder-current').textContent='';$('#folder-search').value='';}
  try{
    const result=await api(`/api/pikpak/folders?${new URLSearchParams({parent_id:parent,token})}`);
    if(sequence!==folderRequest)return;
    if(append&&folderState){if(folderState.account_ref!==result.account_ref)throw new Error('PikPak 帳號已更換，請重新開啟資料夾選擇器');const known=new Set(folderState.folders.map(f=>f.id));result.folders=[...folderState.folders,...result.folders.filter(f=>!known.has(f.id))];}
    folderState=result; renderFolders(); $('#folder-status').textContent=result.next_token?'還有其他項目，按「載入更多資料夾」繼續讀取。':'';
  }catch(e){if(sequence===folderRequest){$('#folder-status').className='field-help error';$('#folder-status').textContent=e.message;}}
  finally{if(sequence===folderRequest)$('#folder-more').disabled=false;}
}
function showRenamePreview(result){
  const element=$('#preview-result');element.className='preview-result success';element.innerHTML=`<div class="rename-preview-row"><span>原始檔名</span><code>${escapeHTML(result.old_name)}</code></div><div class="rename-preview-row"><span>新檔名</span><code>${escapeHTML(result.name)}</code></div>${result.matched?'':'<p class="field-help">Regex 未匹配，保留原名。</p>'}`;
}
async function showJob(id) {
  try { const details=await api(`/api/jobs/${encodeURIComponent(id)}`); const j=details.job;
    const fileLabels={pending:'等待整理',renamed:'已重命名',done:'已完成',review:'保留原名'};
    $('#job-details').innerHTML=`<h3>${escapeHTML(j.rule.title)}</h3><p class="muted">${escapeHTML(j.title)}</p><div class="detail-meta">${badge(j.state)}<span>目標：${escapeHTML(j.destination||'根目錄')}</span></div>${j.error?`<p class="notice">${escapeHTML(j.error)}</p>`:''}<dl class="id-list"><dt>任務識別</dt><dd>${escapeHTML(j.id)}</dd><dt>PikPak 任務 ID</dt><dd>${escapeHTML(j.task_id||'尚未取得')}</dd><dt>專用暫存目錄 ID</dt><dd>${escapeHTML(j.staging_id||'尚未建立')}</dd></dl>${details.files.length?details.files.map(f=>`<article class="file-detail"><span class="badge ${f.state==='done'?'green':'orange'}">${escapeHTML(fileLabels[f.state]||f.state)}</span><p>${escapeHTML(f.original_name)}</p><strong>${escapeHTML(f.target_name||'保留原名')}</strong>${f.error?`<small class="error">${escapeHTML(f.error)}</small>`:''}</article>`).join(''):'<p class="muted">尚無逐檔整理紀錄。</p>'}${reviewStates.includes(j.state)&&j.state!=='failed'?`<div class="dialog-footer"><button class="button primary" data-retry="${j.id}">${j.state==='submission_unknown'?'重新核對結果':'重新整理／接續任務'}</button></div>`:''}`;
    $('#job-dialog').showModal();
  }catch(e){toast(e.message,true);}
}
document.addEventListener('click',async event=>{
  const button=event.target.closest('button'); if(!button)return;
  if(button.dataset.view){showView(button.dataset.view);return;}
  if(button.classList.contains('new-subscription')){openSubscription();return;}
  if(button.classList.contains('close-dialog')){closeFolderBrowser();$('#subscription-dialog').close();return;}
  if(button.classList.contains('close-job')){$('#job-dialog').close();return;}
  if(button.dataset.edit){openSubscription(button.dataset.edit);return;}
  if(button.id==='browse-folders'){await loadFolders($('#sub-destination-id').value);return;}
  if(button.id==='folder-close'){closeFolderBrowser();return;}
  if(button.hasAttribute('data-folder')){await loadFolders(button.dataset.folder);return;}
  if(button.id==='folder-more'&&folderState){await loadFolders(folderState.current.id,folderState.next_token,true);return;}
  if(button.id==='folder-select'&&folderState){$('#sub-destination').value=folderState.current.path;$('#sub-destination-id').value=folderState.current.id;$('#sub-destination-account-ref').value=folderState.account_ref;closeFolderBrowser();return;}
  if(button.dataset.job){await showJob(button.dataset.job);return;}
  if(button.dataset.filter){jobFilter=button.dataset.filter;for(const b of document.querySelectorAll('[data-filter]'))b.classList.toggle('selected',b===button);render();return;}
  button.disabled=true;
  try {
    if(button.id==='folder-create'&&folderState){const name=$('#new-folder-name').value;if(!name.trim())throw new Error('請輸入資料夾名稱');const result=await api('/api/pikpak/folders','POST',{parent_id:folderState.current.id,name,account_ref:folderState.account_ref});$('#new-folder-name').value='';await loadFolders(result.folder.id);toast('資料夾已建立，可按「使用此資料夾」選取');}
    if(button.id==='refresh'){await load();toast('已更新面板');}
    if(button.dataset.check||button.dataset.backfill){const id=button.dataset.check||button.dataset.backfill;if(button.dataset.backfill&&!confirm('補抓 RSS 中現有的項目？符合規則的項目會建立雲端任務並使用 PikPak 配額。'))return;await api(`/api/subscriptions/${id}/check`,'POST',{backfill:!!button.dataset.backfill});toast('訂閱檢查完成');await load();}
    if(button.dataset.delete){if(!confirm('刪除這筆訂閱？已建立的任務與雲端檔案會保留。'))return;await api(`/api/subscriptions/${button.dataset.delete}`,'DELETE');toast('訂閱已刪除');await load();}
    if(button.dataset.retry){await api(`/api/jobs/${button.dataset.retry}/retry`,'POST',{});$('#job-dialog').close();toast('已重新核對或排入接續處理');await load();}
    if(button.id==='check-connection'){await api('/api/settings/pikpak/check','POST',{});toast('PikPak 連線已更新；暫停的任務可個別接續');await load();}
    if(button.id==='logout'||button.id==='logout-mobile'){await api('/api/logout','POST',{});location.reload();}
    if(button.id==='preview-button'){const result=await api('/api/rules/preview','POST',{rule:formRule(),title:$('#preview-title').value,filename:$('#preview-filename').value});showRenamePreview(result);}
  }catch(e){if(button.id==='preview-button'){$('#preview-result').className='preview-result error';$('#preview-result').textContent=e.message;}else toast(e.message,true);}
  finally{button.disabled=false;}
});
async function start(){
  const session=await api('/api/session');csrf=session.csrf;
  if($('#login-form')){
    $('#login-form').addEventListener('submit',async event=>{event.preventDefault();const button=event.target.querySelector('button');button.disabled=true;$('#login-error').textContent='';try{await api('/api/login','POST',{password:$('#password').value});location.reload();}catch(e){$('#login-error').textContent=e.message;}finally{button.disabled=false;}});return;
  }
  $('#subscription-form').addEventListener('submit',async event=>{event.preventDefault();const button=event.target.querySelector('button[type=submit]');button.disabled=true;try{const id=$('#sub-id').value;const rule=formRule();await api(`/api/subscriptions${id?'/'+id:''}`,id?'PUT':'POST',{name:rule.title,rss_url:$('#sub-url').value.trim(),destination:$('#sub-destination').value.trim(),destination_id:$('#sub-destination-id').value,destination_account_ref:$('#sub-destination-account-ref').value,enabled:$('#sub-enabled').checked,interval_minutes:Number($('#sub-interval').value),season:rule.season,regex:rule.regex,template:rule.template,rename_enabled:rule.rename_enabled,rename_mode:rule.mode,replacement:rule.replacement});closeFolderBrowser();$('#subscription-dialog').close();toast('訂閱設定已儲存');await load();}catch(e){toast(e.message,true);}finally{button.disabled=false;}});
  $('#token-form').addEventListener('submit',async event=>{event.preventDefault();const button=$('#save-token');button.disabled=true;try{await api('/api/settings/pikpak','POST',{token:$('#token').value});$('#token').value='';toast('PAT 已驗證並安全保存');await load();}catch(e){$('#token').value='';toast(e.message,true);}finally{button.disabled=!!data.status.external;}});
  $('#subscription-search').addEventListener('input',render);showView(location.hash.slice(1)||'overview');await load();
  $('#sub-rename-enabled').addEventListener('change',updateRenameOptions);$('#sub-rename-mode').addEventListener('change',updateRenameOptions);
  $('#sub-destination').addEventListener('input',()=>{$('#sub-destination-id').value='';$('#sub-destination-account-ref').value='';});
  $('#folder-search').addEventListener('input',renderFolders);
  $('#new-folder-name').addEventListener('keydown',event=>{if(event.key==='Enter'){event.preventDefault();$('#folder-create').click();}});
  setInterval(()=>{if(!document.hidden&&!$('#subscription-dialog').open&&!$('#job-dialog').open)load(true);},15000);
}
start().catch(e=>{if($('#login-error'))$('#login-error').textContent=e.message;else toast(e.message,true);});
