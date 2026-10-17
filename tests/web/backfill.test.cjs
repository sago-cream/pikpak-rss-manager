'use strict';
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');

function harness() {
  const elements=new Map();
  const element=id=>{
    if(!elements.has(id))elements.set(id,{
      disabled:false,checked:false,indeterminate:false,textContent:'',listeners:{},
      addEventListener(type,listener){this.listeners[type]=listener;},
      async dispatch(type){await this.listeners[type]?.({target:this,preventDefault(){}});},
    });
    return elements.get(id);
  };
  let inputs=[],resolvePreview,resolveSubmission,rejectSubmission,confirmation=true;
  const calls=[];
  const list=element('backfill-list');
  list.replaceChildren=()=>{inputs=[];};
  Object.defineProperty(list,'innerHTML',{set(html){
    inputs=Array.from(html.matchAll(/<input type="checkbox" value="([^"]*)" ([^>]*)>/g),match=>({
      value:match[1],disabled:match[2].includes('disabled'),checked:false,
    }));
  }});
  const dialog=element('backfill-dialog');
  dialog.showModal=()=>{dialog.open=true;};
  dialog.close=()=>{dialog.open=false;dialog.dispatch('close');};
  const ctx=vm.createContext({
    AbortController,
    $:selector=>element(selector.slice(1)),
    document:{addEventListener(){},querySelectorAll(selector){
      assert.ok(selector.startsWith('#backfill-list input'));
      return inputs.filter(input=>(!selector.includes(':checked')||input.checked)&&(!selector.includes(':not(:disabled)')||!input.disabled));
    }},
    t:text=>text,escapeHTML:String,confirm:()=>confirmation,toast(){},load:async()=>{},
    api:async(path,method,body,signal)=>{
      calls.push({path,method,body,signal});
      if(path.endsWith('/preview'))return new Promise(resolve=>{resolvePreview=resolve;});
      return new Promise((resolve,reject)=>{resolveSubmission=resolve;rejectSubmission=reject;});
    },
  });
  vm.runInContext(fs.readFileSync('internal/web/static/backfill.js','utf8'),ctx);
  return {
    element,calls,get inputs(){return inputs;},get pendingPreview(){return resolvePreview;},
    open:id=>vm.runInContext('openBackfill('+JSON.stringify(id)+')',ctx),
    preview:items=>resolvePreview({token:'fixture-token',items,notices:[]}),
    confirm:value=>{confirmation=value;},
    succeed:()=>resolveSubmission({count:2}),fail:()=>rejectSubmission(new Error('Fixture failure')),
  };
}
const items=[
  {id:'one',title:'Fixture 01',filenames:['Fixture 01.mkv','Notes.txt']},
  {id:'two',title:'Fixture 02',filenames:['Fixture 02.mkv']},
  {id:'bad',title:'Unavailable fixture',error:'Fixture metadata error'},
];
(async()=>{
  const h=harness(),all=h.element('backfill-select-all'),download=h.element('backfill-download');
  let opened=h.open('fixture');
  assert.equal(all.disabled,true,'No bulk selection while loading');
  assert.equal(download.disabled,true);
  h.preview(items);await opened;
  assert.equal(all.disabled,false);
  assert.equal(all.checked,false);
  assert.equal(all.indeterminate,false);
  assert.equal(download.disabled,true);

  all.checked=true;await all.dispatch('change');
  assert.deepEqual(h.inputs.map(input=>input.checked),[true,true,false],'Select whole torrents, skip unavailable rows');
  assert.equal(all.checked,true);
  assert.equal(all.indeterminate,false);
  assert.equal(download.disabled,false);
  assert.equal(h.calls.length,1,'Selection must not queue downloads');
  h.confirm(false);await h.element('backfill-form').dispatch('submit');h.confirm(true);
  assert.equal(h.calls.length,1,'Canceled confirmation must not queue downloads');
  assert.equal(all.disabled,false);

  h.inputs[0].checked=false;await h.element('backfill-list').dispatch('change');
  assert.equal(all.checked,false);
  assert.equal(all.indeterminate,true);
  assert.equal(download.disabled,false);
  all.checked=true;await all.dispatch('change');
  all.checked=false;await all.dispatch('change');
  assert.ok(h.inputs.every(input=>!input.checked));
  assert.equal(all.indeterminate,false);
  assert.equal(download.disabled,true);
  for(const input of h.inputs.filter(input=>!input.disabled))input.checked=true;
  await h.element('backfill-list').dispatch('change');
  assert.equal(all.checked,true,'Individual selection updates the bulk checkbox');

  const submitting=h.element('backfill-form').dispatch('submit');
  assert.equal(all.disabled,true);
  assert.equal(download.disabled,true);
  assert.ok(h.inputs.every(input=>input.disabled));
  assert.deepEqual(Array.from(h.calls[1].body.selected),['one','two']);
  assert.equal(h.calls[1].body.token,'fixture-token');
  await h.element('backfill-form').dispatch('submit');
  assert.equal(h.calls.length,2,'Ignore duplicate submissions while pending');
  h.fail();await submitting;
  assert.equal(download.disabled,false,'Allow the same confirmation to be retried');
  assert.equal(all.disabled,true,'Keep the submitted selection frozen on failure');
  const retry=h.element('backfill-form').dispatch('submit');
  assert.equal(JSON.stringify(h.calls[2].body),JSON.stringify(h.calls[1].body));
  h.succeed();await retry;
  assert.equal(h.element('backfill-dialog').open,false);

  for(const nextItems of [[],[items[2]],items]){
    opened=h.open('fixture');
    assert.equal(all.disabled,true);
    assert.equal(all.checked,false);
    assert.equal(all.indeterminate,false);
    h.preview(nextItems);await opened;
    assert.equal(all.disabled,nextItems.length<2,'Empty/unavailable plans disable bulk selection');
    assert.equal(download.disabled,true);
  }
  // Late previews must not repopulate a closed dialog or replace a newer plan.
  opened=h.open('old');
  const resolveClosed=h.pendingPreview,oldCall=h.calls.at(-1);
  h.element('backfill-dialog').close();
  resolveClosed({items,notices:[]});await opened;
  assert.equal(oldCall.signal.aborted,true);
  assert.equal(h.inputs.length,0);
  const stale=h.open('stale'),resolveStale=h.pendingPreview;
  opened=h.open('new');h.preview([items[0]]);await opened;
  resolveStale({items,notices:[]});await stale;
  all.checked=true;await all.dispatch('change');
  assert.equal(all.checked,true);
  assert.equal(all.indeterminate,false);
  assert.equal(h.inputs.length,1);
  console.log('Backfill selection checks passed.');
})().catch(error=>{console.error(error);process.exitCode=1;});
