'use strict';
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const source=fs.readFileSync('internal/web/static/app.js','utf8');
const handler=source.slice(source.indexOf('function renderOverview(){'),source.indexOf('function render()'));
let renderedJobs;
const ctx=vm.createContext({t:value=>value,translateLiteral:value=>value,
  tr:(strings,...values)=>strings.reduce((out,part,i)=>out+part+(values[i]??''),''),
  data:{subscriptions:[],jobs:[],status:{}},
  reviewStates:['needs_review','submission_unknown','failed','paused_auth','paused_quota','paused_account'],
  runningStates:['queued','submitting','downloading','organizing'],
  jobsTable:(items,overview)=>{assert.equal(overview,true);renderedJobs=items;return '<table>fixture jobs</table>';}});
vm.runInContext(handler,ctx);
let html=vm.runInContext('renderOverview()',ctx);
assert.match(html,/新增第一筆下載/);
assert.match(html,/data-manual-task/);
assert.match(html,/new-subscription/);
assert.doesNotMatch(html,/data-overview-filter/);
ctx.data.subscriptions=[{id:1}];
html=vm.runInContext('renderOverview()',ctx);
assert.match(html,/目前沒有進行中的任務/);
const job=(id,state)=>({id,state});
ctx.data.jobs=[job('running','downloading'),job('complete','complete'),job('attention','needs_review')];
html=vm.runInContext('renderOverview()',ctx);
assert.deepEqual(Array.from(renderedJobs,item=>item.id),['attention','running']);
assert.match(html,/data-overview-filter="review"/);
assert.match(html,/data-overview-filter="running"/);
ctx.data.jobs=Array.from({length:12},(_,i)=>job(i,'queued'));
vm.runInContext('renderOverview()',ctx);
assert.equal(renderedJobs.length,8);
ctx.data.jobs=Array.from({length:5},(_,i)=>job(i,'complete'));
html=vm.runInContext('renderOverview()',ctx);
assert.equal(renderedJobs.length,3);
assert.doesNotMatch(html,/data-overview-filter/);
ctx.data.status.paused=true;
html=vm.runInContext('renderOverview()',ctx);
assert.match(html,/修復連線/);
assert.match(html,/data-view="settings"/);
console.log('Overview empty state, work priority, bounded history and paused connection checks passed.');
