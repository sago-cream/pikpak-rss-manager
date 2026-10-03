'use strict';
let backfillRequest, backfillPlan;
async function openBackfill(id) {
  const dialog=$('#backfill-dialog');
  backfillRequest?.abort();backfillRequest=new AbortController();backfillPlan=null;
  $('#backfill-list').replaceChildren();$('#backfill-status').textContent=t('正在讀取下載清單…');
  $('#backfill-download').disabled=true;dialog.showModal();
  const request=backfillRequest;
  try {
    const plan=await api(`/api/subscriptions/${id}/backfill/preview`,'POST',{},request.signal);
    if(request!==backfillRequest||!dialog.open)return;
    backfillPlan={...plan,id};
    $('#backfill-list').innerHTML=plan.items.map(item=>`<label class="backfill-choice"><input type="checkbox" value="${escapeHTML(item.id)}" ${item.error?'disabled':''}><span><strong>${escapeHTML(item.title)}</strong>${(item.filenames||[]).map(name=>`<code>${escapeHTML(name)}</code>`).join('')}${!item.filenames?.length?`<small>${escapeHTML(t('無法取得種子檔名'))}</small>`:''}${item.error?`<small class="error">${escapeHTML(item.error)}</small>`:''}</span></label>`).join('');
    $('#backfill-status').textContent=[!plan.items.length?t('RSS 中沒有可下載的項目'):'',...plan.notices].filter(Boolean).join(' ');
  }catch(e){if(request===backfillRequest&&dialog.open)$('#backfill-status').textContent=e.message;}
}
document.addEventListener('click',event=>{
  if(event.target.closest('.close-backfill'))$('#backfill-dialog').close();
});
const backfillDialog=$('#backfill-dialog');
if(backfillDialog){
  backfillDialog.addEventListener('close',()=>{backfillRequest?.abort();backfillRequest=null;backfillPlan=null;});
  $('#backfill-list').addEventListener('change',()=>{
    $('#backfill-download').disabled=!backfillPlan||!$('#backfill-list input:checked');
  });
  $('#backfill-form').addEventListener('submit',async event=>{
    event.preventDefault();if(!backfillPlan)return;
    const plan=backfillPlan,selected=Array.from(document.querySelectorAll('#backfill-list input:checked'),input=>input.value);
    if(!selected.length)return;
    if(!confirm(t('下載所選項目並覆蓋目標資料夾的同名檔案？舊檔會移至任務備份目錄。')))return;
    const button=$('#backfill-download');button.disabled=true;
    for(const input of document.querySelectorAll('#backfill-list input'))input.disabled=true;
    try{
      const result=await api(`/api/subscriptions/${plan.id}/backfill`,'POST',{token:plan.token,selected,overwrite:true});
      backfillDialog.close();toast(t('已排入下載佇列：%d 筆').replace('%d',result.count));await load();
    }catch(e){$('#backfill-status').textContent=e.message;button.disabled=false;}
  });
}
