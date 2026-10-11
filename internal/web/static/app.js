'use strict';
const $ = (selector) => document.querySelector(selector);
const escapeHTML = (value) => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let csrf = '', toastTimer;
async function api(path, method = 'GET', body, signal) {
  const controller = new AbortController();
  const abort = () => controller.abort();
  if(signal?.aborted)abort();else signal?.addEventListener('abort',abort,{once:true});
  const timer = setTimeout(() => controller.abort(), 110000);
  try {
    const response = await fetch(path, {method, credentials: 'same-origin', signal: controller.signal,
      headers: {'Content-Type':'application/json', 'X-CSRF-Token':csrf}, body: body === undefined ? undefined : JSON.stringify(body)});
    const result = localizeResponse(await response.json());
    if (!response.ok) {
      if (response.status === 401 && path !== '/api/login' && !$('#login-form') && !$('#setup-form')) location.reload();
      throw new Error(result.error || t('操作未完成，請稍後再試'));
    }
    return result;
  } catch (error) { if (error.name === 'AbortError') throw new Error(t('操作逾時，請檢查任務紀錄後再試')); throw error; }
  finally { clearTimeout(timer);signal?.removeEventListener('abort',abort); }
}
function toast(message, error = false) {
  const element = $('#toast'); if (!element) return;
  clearTimeout(toastTimer); element.textContent = message; element.className = tr`toast ${error ? 'is-error' : ''}`;
  toastTimer = setTimeout(() => element.classList.add('hidden'), 4500);
}
function date(timestamp) { return timestamp ? new Date(timestamp * 1000).toLocaleString(language === 'en' ? 'en-US' : 'zh-TW', {month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hour12:false}) : t('尚未檢查'); }
const labels = {queued:t('等待處理'),submitting:t('正在提交'),submission_unknown:t('提交待核對'),downloading:t('雲端下載中'),organizing:t('正在整理'),complete:t('已完成'),needs_review:t('待處理'),failed:t('下載失敗'),paused_auth:t('授權暫停'),paused_quota:t('配額暫停'),paused_account:t('帳號暫停')};
const runningStates = ['queued','submitting','downloading','organizing'];
const reviewStates = ['needs_review','submission_unknown','failed','paused_auth','paused_quota','paused_account'];
function badge(state) { return tr`<span class="badge ${state === 'complete' ? 'green' : reviewStates.includes(state) ? 'orange' : 'purple'}">${escapeHTML(labels[state] || state)}</span>`; }
const views = {overview:t('總覽'),subscriptions:t('訂閱管理'),jobs:t('離線任務'),events:t('執行日誌'),settings:t('系統設定')};
let data = {subscriptions:[], jobs:[], events:[], status:{}}, currentView = 'overview', jobFilter = 'all', jobLimit = 8;
function showView(name) {
  if (!views[name]) return;
  currentView = name;
  const dialog=$('#utility-dialog');
  $('#profile-menu').hidePopover();
  if(name==='settings'||name==='events'){
    if(name==='settings')showSettingsSection('pikpak');
    for(const element of document.querySelectorAll('.utility-view'))element.classList.toggle('hidden',element.id!==`view-${name}`);
    $('#utility-title').textContent=views[name];
    if(!dialog.open)dialog.showModal();
  }else{
    if(dialog.open)dialog.close();
    const target=name==='overview'?$('.workspace-header'):$('#view-'+name);
    if(!target.classList.contains('hidden'))target.scrollIntoView({block:'start'});
  }
  history.replaceState(null,'',tr`#${name}`);
}
function showSettingsSection(name){
  if(!['pikpak','site','password'].includes(name))return;
  for(const button of document.querySelectorAll('[data-settings-section]')){
    const selected=button.dataset.settingsSection===name;
    button.setAttribute('aria-current',String(selected));
    $('#'+button.getAttribute('aria-controls')).classList.toggle('hidden',!selected);
  }
}
function filteredJobs(){
  const items=data.jobs.filter(j=>jobFilter==='all'||(jobFilter==='running'&&runningStates.includes(j.state))||(jobFilter==='review'&&reviewStates.includes(j.state))||(jobFilter==='complete'&&j.state==='complete'));
  if(jobFilter==='all'){
    const priority=j=>reviewStates.includes(j.state)?0:runningStates.includes(j.state)?1:2;
    items.sort((a,b)=>priority(a)-priority(b));
  }
  return items;
}
function selectJobFilter(filter){
  if(!['all','running','review','complete'].includes(filter))return;
  const focused=document.activeElement?.id==='job-status-filter';
  jobFilter=filter;jobLimit=8;render();
  if(focused)$('#job-status-filter').focus({preventScroll:true});
}
function icon(name){
  const paths={
    rss:'<path d="M5 11a8 8 0 0 1 8 8M5 4a15 15 0 0 1 15 15"/><circle cx="5" cy="19" r="1"/>',
    folder:'<path d="M3 7h7l2 2h9v11H3zM3 7V4h7l2 3"/>',
    clock:'<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    edit:'<path d="m15 4 5 5-11 11H4v-5zM13 6l5 5"/>',
    check:'<path d="m9 12 2 2 4-4"/><circle cx="12" cy="12" r="9"/>',
    backfill:'<path d="M12 3v12m-4-4 4 4 4-4M4 15v5h16v-5"/>',
    trash:'<path d="M3 6h18M9 6V3h6v3M6 6l1 15h10l1-15M10 10v7m4-7v7"/>'
  };
  return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">${paths[name]}</svg>`;
}
function empty(title, description, button = '') { return tr`<div class="empty-state"><h3>${escapeHTML(title)}</h3>${description?tr`<p>${escapeHTML(description)}</p>`:''}${button}</div>`; }
function subscriptionRows(items) {
  if(!items.length)return tr`<p class="workspace-idle">${data.subscriptions.length?t('找不到訂閱'):t('尚無訂閱')}</p>`;
  return items.map(s=>tr`<article class="subscription-row"><div class="subscription-summary"><span class="subscription-symbol">${icon('rss')}</span><div class="subscription-info"><div class="subscription-name"><h3>${escapeHTML(s.name)}</h3><span class="badge ${s.enabled?'green':''}">${s.enabled?t('追蹤中'):t('已停用')}</span></div><div class="subscription-meta"><span title="${escapeHTML(s.destination||t('根目錄'))}">${icon('folder')}<span class="destination-text">${escapeHTML(s.destination||t('根目錄'))}</span></span><span>${icon('clock')}每 ${s.interval_minutes} 分鐘</span></div></div></div>${s.last_error?tr`<p class="error">${escapeHTML(s.last_error)}</p>`:''}<div class="subscription-footer"><time class="muted" title="${t('上次檢查')}">${date(s.last_checked)}</time><div class="subscription-actions"><button class="icon-button" data-edit="${s.id}" aria-label="編輯訂閱" title="編輯訂閱">${icon('edit')}</button><button class="icon-button" data-check="${s.id}" aria-label="檢查新的發布" title="檢查新的發布">${icon('check')}</button><button class="icon-button" data-backfill="${s.id}" aria-label="補抓" title="處理 RSS 中仍可取得的現有項目">${icon('backfill')}</button><button class="icon-button danger" data-delete="${s.id}" aria-label="刪除 ${escapeHTML(s.name)}" title="刪除訂閱">${icon('trash')}</button></div></div></article>`).join('');
}
function jobsTable(items) {
  const filters={all:t('全部'),running:t('處理中'),review:t('待處理'),complete:t('已完成')};
  const options=Object.entries(filters).map(([value,label])=>`<option value="${value}"${jobFilter===value?' selected':''}>${t('狀態')} · ${label}</option>`).join('');
  const rows=items.map(j=>tr`<tr><td class="job-name"><strong>${escapeHTML(j.rule.title)}</strong><span title="${escapeHTML(j.title)}">${escapeHTML(j.title)}</span>${j.error ? tr`<small class="error">${escapeHTML(j.error)}</small>` : ''}</td><td>${badge(j.state)}${j.state==='downloading' ? tr`<div class="progress-caption">${Math.round(j.progress)}%</div>` : ''}</td><td class="muted nowrap">${date(j.created_at)}</td><td><div class="job-actions"><button class="text-button" data-job="${escapeHTML(j.id)}">查看 →</button></div></td></tr>`).join('');
  return tr`<div class="table-scroll"><table><thead><tr><th scope="col">發布與作品</th><th scope="col" class="job-status-column"><select id="job-status-filter" class="job-status-filter" aria-label="任務狀態">${options}</select></th><th scope="col">時間</th><th scope="col"></th></tr></thead><tbody>${rows||tr`<tr class="jobs-empty"><td colspan="4"><p class="workspace-idle">${data.jobs.length?t('沒有符合的任務'):t('尚無任務')}</p></td></tr>`}</tbody></table></div>`;
}
function renderOverview(){
  const paused=data.status.paused?translateLiteral('<div class="overview-connection"><span>PikPak 已暫停</span><button class="text-button" data-view="settings">修復連線 →</button></div>'):'';
  if(data.subscriptions.length||data.jobs.length)return paused;
  return paused+translateLiteral('<div class="overview-empty"><h2>新增第一筆下載</h2><p>訂閱 RSS 自動追蹤，或貼上連結下載。</p><div class="form-actions overview-actions"><button class="button primary new-subscription">＋ 新增訂閱</button><button class="button subtle" data-manual-task>＋ 新增任務</button></div></div>');
}
function render() {
  $('#overview-content').innerHTML=renderOverview();
  const isEmpty=!data.subscriptions.length&&!data.jobs.length;
  for(const element of document.querySelectorAll('.workspace-section, .heading-actions [data-manual-task], .heading-actions .new-subscription'))element.classList.toggle('hidden',isEmpty);
  const search=($('#subscription-search').value||'').toLowerCase();
  $('#subscriptions-list').innerHTML=subscriptionRows(data.subscriptions.filter(s=>s.name.toLowerCase().includes(search)));
  $('#subscription-count').textContent=tr`共 ${data.subscriptions.length} 筆訂閱`;
  const jobs=filteredJobs();
  $('#jobs-list').innerHTML=jobsTable(jobs.slice(0,jobLimit));
  $('#jobs-more').classList.toggle('hidden',jobs.length<=jobLimit);
  $('#clear-completed-jobs').classList.toggle('hidden',jobFilter!=='complete'||!jobs.length);
  $('#events-list').innerHTML=data.events.length ? data.events.map(e=>tr`<div class="event-row"><span class="event-dot ${escapeHTML(e.level)}"></span><div><p>${escapeHTML(e.message)}</p><small class="muted">${e.job_id ? tr`任務 ${escapeHTML(e.job_id.slice(0,8))}` : e.subscription_id ? tr`訂閱 #${e.subscription_id}` : t('系統')}</small></div><time>${date(e.created_at)}</time></div>`).join('') : empty(t('尚無日誌'),'');
  const status=data.status;
  const connected=status.connected&&!status.paused;
  $('#profile-connection').innerHTML=tr`<span class="dot ${connected?'green':'amber'}"></span>${connected?t('PikPak 已連線'):status.paused?t('PikPak 已暫停'):t('尚未綁定 PikPak')}`;
  $('#profile-name').textContent=status.name||t('PikPak 帳號');
  $('#settings-badge').className=tr`badge ${connected?'green':'orange'}`;
  $('#settings-badge').textContent=connected?t('連線正常'):status.paused?t('已暫停'):t('未綁定');
  $('#settings-connection-status').textContent=connected?'':status.message||t('請重新檢查 PikPak 連線。');
  $('#account-details').innerHTML=status.connected?tr`<div class="account-card"><span class="account-avatar">P</span><div><strong>${escapeHTML(status.name||t('PikPak 帳號'))}</strong><span>${formatBytes(status.storage_used)} / ${formatBytes(status.storage_total)} 雲端空間</span></div></div>`:'';
}
function formatBytes(value) { let n=Number(value||0); if (!Number.isFinite(n)) return '—'; const units=['B','KiB','MiB','GiB','TiB']; let i=0; while(n>=1024&&i<4){n/=1024;i++;} return tr`${n.toFixed(i?1:0)} ${units[i]}`; }
async function load(silent=false) { try { const [subscriptions,jobs,events,status]=await Promise.all(['/api/subscriptions','/api/jobs','/api/events','/api/settings/pikpak'].map(p=>api(p))); data={subscriptions,jobs,events,status}; render(); } catch(e){if(!silent)toast(e.message,true);} }
let manualTask=false;
function openSubscription(id, manual=false) {
  manualTask=manual;
  const sub=data.subscriptions.find(s=>s.id===Number(id)); $('#subscription-form').reset();
  $('#sub-id').value=sub?.id||''; $('#dialog-title').textContent=manual?t('新增任務'):sub?t('編輯訂閱'):t('新增訂閱');
  $('#sub-name-field').classList.toggle('hidden',manual);
  $('#sub-name').required=!manual;
  $('#sub-name').disabled=manual;
  $('#sub-url-label').textContent=manual?t('下載連結'):t('RSS 連結');
  $('#sub-url').type=manual?'text':'url';
  $('#sub-url').placeholder=manual?'magnet:?xt=… / https://…':'https://…';
  for(const id of ['interval-field','enabled-field','rename-toggle','baseline-help'])$('#'+id).classList.toggle('hidden',manual);
  $('#save-subscription').textContent=manual?t('新增任務'):t('儲存訂閱');
  $('#sub-name').value=sub?.name||''; $('#sub-url').value=sub?.rss_url||''; $('#sub-destination').value=sub?.destination||'';
  $('#sub-destination-id').value=sub?.destination_id||''; $('#sub-destination-account-ref').value=sub?.destination_account_ref||'';
  $('#sub-interval').value=sub?.interval_minutes||10; $('#sub-enabled').checked=sub?.enabled??true;
  $('#sub-rename-enabled').checked=sub?.rename_enabled??false;
  $('#sub-regex').value=sub?.regex??'\\[(\\d+)\\]'; $('#sub-replacement').value=sub?.replacement??'S01E$1';
  closeFolderBrowser(); clearSourceSamples(); updateRenameOptions(); $('#browse-folders').disabled=!data.status.connected||!!data.status.paused;
  $('#destination-help').textContent=data.status.connected?'':t('綁定 PikPak PAT 後即可列出或建立資料夾；也可先手動輸入路徑。');
  $('#subscription-dialog').showModal();
}
function formRule(){return {title:$('#sub-name').value.trim(),regex:$('#sub-regex').value,rename_enabled:$('#sub-rename-enabled').checked,replacement:$('#sub-replacement').value};}
function updateRenameOptions(){
  const enabled=$('#sub-rename-enabled').checked;
  $('#rename-options').classList.toggle('hidden',!enabled);
  for(const id of ['sub-regex','sub-replacement'])$('#'+id).disabled=!enabled;
  $('#sub-regex').required=enabled;
  scheduleRenamePreview();
}

let folderState=null, folderRequest=0, folderCreateContext=null;
function closeFolderBrowser(){folderRequest++;folderState=null;folderCreateContext=null;$('#folder-create-dialog').close();$('#folder-browser').classList.add('hidden');$('#folder-list').innerHTML='';}
function openFolderCreate(){
  if(!folderState)return;
  folderCreateContext={parent_id:folderState.current.id,account_ref:folderState.account_ref};
  $('#folder-create-form').reset();$('#folder-create-error').textContent='';$('#folder-create-location').textContent=tr`建立位置：${folderState.current.path||t('根目錄')}`;
  $('#folder-create-form button[type=submit]').disabled=false;
  $('#folder-create-dialog').showModal();
}
function renderFolders(){
  if(!folderState)return;
  $('#folder-breadcrumbs').innerHTML=folderState.breadcrumbs.map(d=>tr`<button type="button" class="text-button" data-folder="${escapeHTML(d.id)}">${escapeHTML(d.id?d.name:t(d.name))}</button>`).join('<span>/</span>');
  $('#folder-search').classList.toggle('hidden',folderState.folders.length<=8&&!folderState.next_token);
  const term=$('#folder-search').value.toLowerCase(), folders=folderState.folders.filter(f=>f.name.toLowerCase().includes(term));
  $('#folder-list').innerHTML=folders.length?folders.map(f=>tr`<button type="button" class="folder-row" data-folder="${escapeHTML(f.id)}"><img class="folder-icon" src="/static/folder.svg" alt=""><span>${escapeHTML(f.name)}</span><span class="folder-arrow">›</span></button>`).join(''):translateLiteral('<p class="field-help">目前沒有符合的資料夾。</p>');
  $('#folder-more').classList.toggle('hidden',!folderState.next_token);
  $('#folder-select').textContent=folderState.current.id?t('使用此資料夾'):t('使用根目錄'); $('#folder-select').disabled=false; $('#folder-create').disabled=false;
}
async function loadFolders(parent='',token='',append=false){
  const sequence=++folderRequest; $('#folder-browser').classList.remove('hidden'); $('#folder-status').className='field-help'; $('#folder-status').textContent=t('正在讀取 PikPak 資料夾…');
  $('#folder-select').disabled=true; $('#folder-create').disabled=true; $('#folder-more').disabled=true;
  if(!append){folderState=null;$('#folder-list').innerHTML='';$('#folder-breadcrumbs').innerHTML='';$('#folder-search').classList.add('hidden');$('#folder-search').value='';}
  try{
    const result=await api(tr`/api/pikpak/folders?${new URLSearchParams({parent_id:parent,token})}`);
    if(sequence!==folderRequest)return;
    if(append&&folderState){if(folderState.account_ref!==result.account_ref)throw new Error(t('PikPak 帳號已更換，請重新開啟資料夾選擇器'));const known=new Set(folderState.folders.map(f=>f.id));result.folders=[...folderState.folders,...result.folders.filter(f=>!known.has(f.id))];}
    folderState=result; renderFolders(); $('#folder-status').textContent='';
  }catch(e){if(sequence===folderRequest){$('#folder-status').className='field-help error';$('#folder-status').textContent=e.message;}}
  finally{if(sequence===folderRequest)$('#folder-more').disabled=false;}
}
function showRenamePreview(result){
  const adjusted=result.raw_name!==undefined&&result.raw_name!==result.name;
  const element=$('#preview-result');element.className='preview-result success';element.innerHTML=tr`<div class="rename-preview-row"><span>替換結果</span><code>${escapeHTML(result.name)}</code></div>${adjusted?tr`<p class="preview-warning">${escapeHTML((result.warnings||[]).join(' '))}</p>`:''}`;
}
let sourceSamples=[],sourceRequest=0,sourceController;
function selectedSourceSample(){const value=$('#preview-source').value;return value===''?undefined:sourceSamples[Number(value)];}
function clearSourceSamples(){sourceRequest++;sourceController?.abort();clearTimeout(previewTimer);previewRequest++;sourceSamples=[];$('#source-samples').disabled=false;$('#source-samples').textContent=t('讀取種子檔名');$('#source-samples').setAttribute('aria-busy','false');$('#preview-source').innerHTML='';$('#preview-source-label').classList.add('hidden');$('#preview-source-help').textContent='';$('#preview-result').className='preview-result';$('#preview-result').textContent=t('選擇種子檔名');}
function selectSourceSample(){
  scheduleRenamePreview();
}
async function loadSourceSamples(){
  const url=$('#sub-url').value.trim();if(!url)throw new Error(t('請先填寫 RSS 連結'));
  const sequence=++sourceRequest;
  sourceController?.abort();const controller=new AbortController();sourceController=controller;
  $('#source-samples').disabled=true;$('#source-samples').textContent=t('讀取中…');$('#source-samples').setAttribute('aria-busy','true');$('#preview-source-help').textContent=t('正在讀取全部種子檔名…');
  try{
    const result=await api('/api/feeds/samples','POST',{url,subscription_id:Number($('#sub-id').value)||0},controller.signal);
    if(sequence!==sourceRequest||!$('#subscription-dialog').open||$('#sub-url').value.trim()!==url)return;
    const selected=selectedSourceSample();sourceSamples=[];
    const known=new Set();
    for(const sample of result.items.filter(s=>s.kind==='torrent_file')){const key=JSON.stringify([sample.title,sample.filename]);if(!known.has(key)){sourceSamples.push(sample);known.add(key);}}
    $('#preview-source').innerHTML=sourceSamples.map((s,i)=>tr`<option value="${i}">${escapeHTML(s.filename)}</option>`).join('');
    $('#preview-source-label').classList.toggle('hidden',!sourceSamples.length);
    const selectedIndex=selected?sourceSamples.findIndex(s=>s.title===selected.title&&s.filename===selected.filename):-1;
    $('#preview-source').value=sourceSamples.length?String(selectedIndex>=0?selectedIndex:0):'';
    selectSourceSample();
    $('#preview-source-help').textContent=(sourceSamples.length?tr`已讀取 ${sourceSamples.length} 個種子檔名。`:t('目前 RSS 沒有可用的種子檔名。'))+(result.notices?.length?tr` ${[...new Set(result.notices)].join(' ')}`:'');
  }catch(e){if(sequence===sourceRequest)$('#preview-source-help').textContent=e.message;}
  finally{if(sequence===sourceRequest){$('#source-samples').disabled=false;$('#source-samples').textContent=t('讀取種子檔名');$('#source-samples').setAttribute('aria-busy','false');}}
}
let previewRequest=0,previewTimer;
function scheduleRenamePreview(event){
  clearTimeout(previewTimer);previewRequest++;
  const element=$('#preview-result'),filename=selectedSourceSample()?.filename;
  element.className='preview-result';element.textContent=filename?t('正在更新預覽…'):t('選擇種子檔名');
  if(!$('#subscription-dialog').open||!$('#sub-rename-enabled').checked||!filename||event?.isComposing)return;
  previewTimer=setTimeout(previewRule,200);
}
async function previewRule(){
  const sample=selectedSourceSample();if(!sample)return;
  const sequence=++previewRequest;
  try{const result=await api('/api/rules/preview','POST',{rule:formRule(),filename:sample.filename});if(sequence===previewRequest&&$('#subscription-dialog').open)showRenamePreview(result);}
  catch(e){if(sequence===previewRequest){$('#preview-result').className='preview-result error';$('#preview-result').textContent=e.message;}}
}
async function showJob(id) {
  try { const details=await api(tr`/api/jobs/${encodeURIComponent(id)}`); const j=details.job;
    const fileLabels={pending:t('等待整理'),renaming:t('改名中'),done:t('已完成'),review:t('保留原名')};
    const files=details.files.map(f=>{
      const currentName=f.actual_name||f.original_name||f.target_name||t('保留原名');
      const original=f.original_name&&f.original_name!==currentName?tr`<p>原名：${escapeHTML(f.original_name)}</p>`:'';
      return tr`<article class="file-detail"><span class="badge ${f.state==='done'?'green':'orange'}">${escapeHTML(fileLabels[f.state]||f.state)}</span>${original}<strong>${escapeHTML(currentName)}</strong>${f.error?tr`<small class="error">${escapeHTML(f.error)}</small>`:''}</article>`;
    }).join('');
    $('#job-details').innerHTML=tr`<h3>${escapeHTML(j.rule.title)}</h3><p class="muted">${escapeHTML(j.title)}</p><div class="detail-meta">${badge(j.state)}<span>目標：${escapeHTML(j.destination||t('根目錄'))}</span></div>${j.error?tr`<p class="notice">${escapeHTML(j.error)}</p>`:''}<dl class="id-list"><dt>任務識別</dt><dd>${escapeHTML(j.id)}</dd><dt>PikPak 任務 ID</dt><dd>${escapeHTML(j.task_id||t('尚未取得'))}</dd></dl>${files||translateLiteral('<p class="muted">尚無逐檔整理紀錄。</p>')}<div class="dialog-footer"><button class="button subtle danger" data-delete-job="${escapeHTML(j.id)}">刪除任務</button>${reviewStates.includes(j.state)&&j.state!=='failed'?tr`<button class="button primary" data-retry="${j.id}">${j.state==='submission_unknown'?t('重新核對結果'):t('重新整理／接續任務')}</button>`:''}</div>`;
    $('#job-dialog').showModal();
  }catch(e){toast(e.message,true);}
}
document.addEventListener('click',async event=>{
  const button=event.target.closest('button'); if(!button)return;
  if(button.dataset.settingsSection){showSettingsSection(button.dataset.settingsSection);return;}
  if(button.dataset.view){showView(button.dataset.view);return;}
  if(button.hasAttribute('data-manual-task')){openSubscription(undefined,true);return;}
  if(button.classList.contains('new-subscription')){openSubscription();return;}
  if(button.classList.contains('close-dialog')){closeFolderBrowser();$('#subscription-dialog').close();return;}
  if(button.classList.contains('close-utility')){$('#utility-dialog').close();return;}
  if(button.id==='jobs-more'){jobLimit+=8;render();return;}
  if(button.classList.contains('close-job')){$('#job-dialog').close();return;}
  if(button.id==='source-samples'){try{await loadSourceSamples();}catch(e){toast(e.message,true);}return;}
  if(button.dataset.edit){openSubscription(button.dataset.edit);return;}
  if(button.id==='browse-folders'){await loadFolders($('#sub-destination-id').value);return;}
  if(button.id==='folder-close'){closeFolderBrowser();return;}
  if(button.id==='folder-create'){openFolderCreate();return;}
  if(button.classList.contains('close-folder-create')){folderCreateContext=null;$('#folder-create-dialog').close();return;}
  if(button.hasAttribute('data-folder')){await loadFolders(button.dataset.folder);return;}
  if(button.id==='folder-more'&&folderState){await loadFolders(folderState.current.id,folderState.next_token,true);return;}
  if(button.id==='folder-select'&&folderState){$('#sub-destination').value=folderState.current.path;$('#sub-destination-id').value=folderState.current.id;$('#sub-destination-account-ref').value=folderState.account_ref;closeFolderBrowser();return;}
  if(button.dataset.job){await showJob(button.dataset.job);return;}
  button.disabled=true;
  try {
    if(button.id==='refresh'){await load();toast(t('已更新'));}
    if(button.dataset.deleteJob){const job=data.jobs.find(j=>j.id===button.dataset.deleteJob);const localOnly=job&&!job.task_id&&(job.submission_pending||['submitting','submission_unknown'].includes(job.state));if(!confirm(localOnly?t('尚無 PikPak 任務 ID，無法自動取消。請先至 PikPak 取消，再刪除此紀錄？'):t('刪除這筆任務並取消 PikPak 下載？已完成的雲端檔案會保留。')))return;await api(`/api/jobs/${encodeURIComponent(button.dataset.deleteJob)}`,'DELETE',{local_only:!!localOnly});$('#job-dialog').close();toast(t('任務已刪除'));await load();}
    if(button.id==='clear-completed-jobs'){if(!confirm(t('清除所有已完成任務紀錄？雲端檔案會保留。')))return;await api('/api/jobs/completed','DELETE');toast(t('已清除已完成任務'));await load();}
    if(button.dataset.backfill){await openBackfill(button.dataset.backfill);}
    if(button.dataset.check){await api(`/api/subscriptions/${button.dataset.check}/check`,'POST',{backfill:false});toast(t('訂閱檢查完成'));await load();}
    if(button.dataset.delete){if(!confirm(t('刪除這筆訂閱？已建立的任務與雲端檔案會保留。')))return;await api(tr`/api/subscriptions/${button.dataset.delete}`,'DELETE');toast(t('訂閱已刪除'));await load();}
    if(button.dataset.retry){await api(tr`/api/jobs/${button.dataset.retry}/retry`,'POST',{});$('#job-dialog').close();toast(t('已重新核對或排入接續處理'));await load();}
    if(button.id==='check-connection'){await api('/api/settings/pikpak/check','POST',{});toast(t('PikPak 連線已更新；暫停的任務可個別接續'));await load();}
    if(button.id==='logout'){await api('/api/logout','POST',{});location.reload();}
  }catch(e){toast(e.message,true);}
  finally{button.disabled=false;}
});
function bindTokenForm(){
  const form=$('#token-form'),onboarding=form.dataset.onboarding==='true';
  form.addEventListener('submit',async event=>{
    event.preventDefault();const button=$('#save-token');button.disabled=true;$('#token-error').textContent='';
    try{
      await api('/api/settings/pikpak','POST',{token:$('#token').value});$('#token').value='';
      if(onboarding){location.hash='overview';location.reload();return;}
      toast(t('PAT 已綁定'));await load();
    }catch(e){$('#token').value='';$('#token-error').textContent=e.message;}
    finally{button.disabled=false;}
  });
}
function bindHelpPopovers(){
  for(const popover of document.querySelectorAll('.field-popover')){
    popover.addEventListener('toggle',event=>{
      if(event.newState!=='open')return;
      const trigger=document.querySelector(`[popovertarget="${popover.id}"]`),rect=trigger.getBoundingClientRect();
      popover.style.left=Math.max(16,Math.min(rect.left,innerWidth-popover.offsetWidth-16))+'px';
      popover.style.top=(rect.bottom+8+popover.offsetHeight<=innerHeight-16?rect.bottom+8:Math.max(16,rect.top-popover.offsetHeight-8))+'px';
    });
  }
}
function bindRecoveryForm(){
  const form=$('#recovery-form');
  form.addEventListener('submit',async event=>{
    event.preventDefault();const button=form.querySelector('button[type=submit]');button.disabled=true;$('#recovery-error').textContent='';
    try{
      const password=$('#recovery-password').value,confirm=$('#recovery-confirm').value;
      if(password!==confirm)throw new Error(t('兩次輸入的密碼不一致'));
      await api('/api/password/recovery','POST',{code:$('#recovery-code').value.trim(),new_password:password,confirm_password:confirm});
      form.reset();form.classList.add('hidden');$('#recovery-help').classList.add('hidden');
      $('#recovery-title').textContent=t('密碼已重設');$('#recovery-success').classList.remove('hidden');
    }catch(error){form.reset();$('#recovery-error').textContent=error.message;}
    finally{button.disabled=false;}
  });
}
async function start(){
  bindHelpPopovers();
  const session=await api('/api/session');csrf=session.csrf;
  if($('#recovery-form')){bindRecoveryForm();return;}
  if($('#setup-form')){
    if(session.initialized){location.reload();return;}
    $('#setup-public-url').value=location.origin;
    $('#setup-form').addEventListener('submit',async event=>{
      event.preventDefault();const button=event.target.querySelector('button[type=submit]');button.disabled=true;$('#setup-error').textContent='';
      try{
        const password=$('#setup-password').value,confirm=$('#setup-confirm').value;
        if(password!==confirm)throw new Error(t('兩次輸入的密碼不一致'));
        await api('/api/setup','POST',{password,confirm_password:confirm,public_url:$('#setup-public-url').value.trim(),allow_private_feeds:$('#setup-private-feeds').checked});
        $('#setup-password').value='';$('#setup-confirm').value='';location.hash='';location.reload();
      }catch(e){$('#setup-password').value='';$('#setup-confirm').value='';$('#setup-error').textContent=e.message;}
      finally{button.disabled=false;}
    });return;
  }
  if($('#login-form')){
    $('#login-form').addEventListener('submit',async event=>{event.preventDefault();const button=event.target.querySelector('button');button.disabled=true;$('#login-error').textContent='';try{await api('/api/login','POST',{password:$('#password').value});$('#password').value='';location.reload();}catch(e){$('#password').value='';$('#login-error').textContent=e.message;}finally{button.disabled=false;}});return;
  }
  bindTokenForm();
  if($('#token-form').dataset.onboarding==='true')return;
  $('#subscription-form').addEventListener('submit',async event=>{event.preventDefault();const button=event.target.querySelector('button[type=submit]');button.disabled=true;try{const id=$('#sub-id').value;const rule=formRule();if(manualTask){await api('/api/jobs','POST',{url:$('#sub-url').value.trim(),destination:$('#sub-destination').value.trim(),destination_id:$('#sub-destination-id').value,destination_account_ref:$('#sub-destination-account-ref').value});}else await api(tr`/api/subscriptions${id?'/'+id:''}`,id?'PUT':'POST',{name:rule.title,rss_url:$('#sub-url').value.trim(),destination:$('#sub-destination').value.trim(),destination_id:$('#sub-destination-id').value,destination_account_ref:$('#sub-destination-account-ref').value,enabled:$('#sub-enabled').checked,interval_minutes:Number($('#sub-interval').value),regex:rule.regex,rename_enabled:rule.rename_enabled,replacement:rule.replacement});closeFolderBrowser();$('#subscription-dialog').close();toast(manualTask?t('任務已新增'):t('訂閱設定已儲存'));await load();}catch(e){toast(e.message,true);}finally{button.disabled=false;}});
  $('#folder-create-form').addEventListener('submit',async event=>{
    event.preventDefault();if(!folderCreateContext)return;
    const context=folderCreateContext,sequence=folderRequest,button=event.target.querySelector('button[type=submit]');
    button.disabled=true;$('#folder-create-error').textContent='';
    try{const result=await api('/api/pikpak/folders','POST',{...context,name:$('#new-folder-name').value});if(context!==folderCreateContext||sequence!==folderRequest||!$('#subscription-dialog').open)return;$('#folder-create-dialog').close();await loadFolders(result.folder.id);toast(t('資料夾已建立，可按「使用此資料夾」選取'));}
    catch(e){if(context===folderCreateContext)$('#folder-create-error').textContent=e.message;}
    finally{if(context===folderCreateContext||!$('#folder-create-dialog').open)button.disabled=false;}
  });
  $('#folder-create-dialog').addEventListener('cancel',()=>{folderCreateContext=null;});
  $('#subscription-dialog').addEventListener('close',()=>{closeFolderBrowser();clearSourceSamples();});
  $('#password-form').addEventListener('submit',async event=>{
    event.preventDefault();const form=event.target,button=form.querySelector('button[type=submit]');button.disabled=true;$('#password-error').textContent='';
    try{
      const password=$('#new-password').value,confirm=$('#confirm-new-password').value;
      if(password!==confirm)throw new Error(t('兩次輸入的密碼不一致'));
      await api('/api/settings/password','POST',{current_password:$('#current-password').value,new_password:password,confirm_password:confirm});
      form.reset();location.hash='';location.reload();
    }catch(e){form.reset();$('#password-error').textContent=e.message;}
    finally{button.disabled=false;}
  });
  const settings=await api('/api/settings/app');$('#settings-public-url').value=settings.public_url;$('#settings-private-feeds').checked=settings.allow_private_feeds;
  $('#app-settings-form').addEventListener('submit',async event=>{event.preventDefault();const button=event.target.querySelector('button[type=submit]');button.disabled=true;try{await api('/api/settings/app','POST',{public_url:$('#settings-public-url').value.trim(),allow_private_feeds:$('#settings-private-feeds').checked});csrf=(await api('/api/session')).csrf;toast(t('網站設定已儲存'));}catch(e){toast(e.message,true);}finally{button.disabled=false;}});
  $('#utility-dialog').addEventListener('close',()=>{if(currentView==='settings'||currentView==='events'){currentView='overview';history.replaceState(null,'','#overview');$('#profile-toggle').focus({preventScroll:true});}});
  $('#profile-menu').addEventListener('beforetoggle',event=>{if(event.newState==='open'){const rect=$('#profile-toggle').getBoundingClientRect(),menu=$('#profile-menu');menu.style.top=(rect.bottom+8)+'px';menu.style.right=Math.max(16,innerWidth-rect.right)+'px';}});
  document.addEventListener('change',event=>{if(event.target.id==='job-status-filter')selectJobFilter(event.target.value);});
  $('#subscription-search').addEventListener('input',render);await load();showView(location.hash.slice(1)||'overview');
  $('#sub-rename-enabled').addEventListener('change',updateRenameOptions);
  for(const id of ['sub-name','sub-regex','sub-replacement']){
    $('#'+id).addEventListener('input',scheduleRenamePreview);$('#'+id).addEventListener('compositionend',scheduleRenamePreview);
  }
  $('#sub-destination').addEventListener('input',()=>{$('#sub-destination-id').value='';$('#sub-destination-account-ref').value='';});
  $('#folder-search').addEventListener('input',renderFolders);
  $('#sub-url').addEventListener('input',clearSourceSamples);$('#preview-source').addEventListener('change',selectSourceSample);
  setInterval(()=>{if(!document.hidden&&!$('#subscription-dialog').open&&!$('#job-dialog').open)load(true);},15000);
}
start().catch(e=>{if($('#setup-error'))$('#setup-error').textContent=e.message;else if($('#login-error'))$('#login-error').textContent=e.message;else if($('#token-error'))$('#token-error').textContent=e.message;else if($('#recovery-error'))$('#recovery-error').textContent=e.message;else toast(e.message,true);});
