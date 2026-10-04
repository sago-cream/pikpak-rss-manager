'use strict';
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const source=fs.readFileSync('internal/web/static/app.js','utf8');
const start=source.indexOf('async function showJob(');
const end=source.indexOf('\n}',start)+2;
let details;
const panel={innerHTML:''};
const dialog={showModal(){}};
const document={documentElement:{},createTreeWalker:()=>({nextNode:()=>false}),querySelectorAll:()=>[]};
const ctx=vm.createContext({document,NodeFilter:{SHOW_TEXT:4},localStorage:{getItem:()=> 'en'},
  $:selector=>selector==='#job-details'?panel:dialog,
  api:async()=>details,escapeHTML:value=>String(value),badge:()=>'',reviewStates:[],toast:message=>{throw Error(message);}});
vm.runInContext(fs.readFileSync('internal/web/static/i18n.js','utf8'),ctx);
vm.runInContext(source.slice(start,end),ctx);
(async()=>{
  for(const language of ['en','zh-Hant']){
    vm.runInContext('language='+JSON.stringify(language),ctx);
    details={job:{id:'fixture',rule:{title:'Fixture'},title:'Fixture',destination:'Downloads',state:'complete',task_id:'fixture-task'},
      files:[{state:'done',original_name:'old.mkv',target_name:'new.mkv',actual_name:'new (1).mkv'}]};
    await vm.runInContext("showJob('fixture')",ctx);
    assert.ok(!panel.innerHTML.includes('Staging folder ID')&&!panel.innerHTML.includes('專用暫存目錄 ID'));
    assert.ok(panel.innerHTML.includes('<strong>new (1).mkv</strong>'));
    // Existing persisted errors receive the new guidance in either language.
    const oldError='改名結果無法確認，保留目前檔名；請核對後重試';
    details.job.state='needs_review';
    details.files=[{state:'review',original_name:oldError,target_name:'new.mkv',actual_name:oldError,error:oldError}];
    details=vm.runInContext('localizeResponse('+JSON.stringify(details)+')',ctx);
    await vm.runInContext("showJob('fixture')",ctx);
    assert.ok(panel.innerHTML.includes('<strong>'+oldError+'</strong>'),'User filenames must remain unchanged');
    assert.ok(panel.innerHTML.includes(language==='en'?'PikPak rejects renaming to a duplicate filename in the same folder.':'PikPak 不允許改成同資料夾內的重複檔名'));
    assert.ok(!details.files[0].error.includes(language==='en'?'Current filename retained; check it before retrying.':oldError));
  }
  console.log('Direct task detail checks passed.');
})().catch(error=>{console.error(error);process.exitCode=1;});
