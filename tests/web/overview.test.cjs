'use strict';
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const source=fs.readFileSync('internal/web/static/app.js','utf8');
const ctx=vm.createContext({t:value=>value,translateLiteral:value=>value,
  data:{subscriptions:[],jobs:[],status:{}},jobFilter:'all',
  reviewStates:['needs_review','submission_unknown','failed','paused_auth','paused_quota','paused_account'],
  runningStates:['queued','submitting','downloading','organizing']});
vm.runInContext(source.slice(source.indexOf('function filteredJobs(){'),source.indexOf('function selectJobFilter('))+source.slice(source.indexOf('function renderOverview(){'),source.indexOf('function render()')),ctx);
let html=vm.runInContext('renderOverview()',ctx);
assert.match(html,/新增第一筆下載/);assert.match(html,/data-manual-task/);assert.match(html,/new-subscription/);
const job=(id,state)=>({id,state});
ctx.data.jobs=[job('running','downloading'),job('complete','complete'),job('attention','needs_review'),job('queued','queued')];
assert.deepEqual(Array.from(vm.runInContext('filteredJobs()',ctx),item=>item.id),['attention','running','queued','complete']);
assert.deepEqual(ctx.data.jobs.map(item=>item.id),['running','complete','attention','queued'],'Priority sorting must not mutate server records');
for(const [filter,ids] of [['running',['running','queued']],['review',['attention']],['complete',['complete']]]){
  ctx.jobFilter=filter;
  assert.deepEqual(Array.from(vm.runInContext('filteredJobs()',ctx),item=>item.id),ids);
}
ctx.data.subscriptions=[{id:1}];ctx.data.jobs=[];
assert.equal(vm.runInContext('renderOverview()',ctx),'','Existing subscriptions stay in the workspace, without a duplicate overview panel');
ctx.data.status.paused=true;html=vm.runInContext('renderOverview()',ctx);
assert.match(html,/修復連線/);assert.match(html,/data-view="settings"/);
console.log('Workspace empty state, task priority, filters and paused connection checks passed.');
// Legacy links reach an in-page section or a utility dialog without hiding the workspace.
const elements=new Map();
const element=selector=>{
  if(!elements.has(selector))elements.set(selector,{id:selector.slice(1),hidden:false,open:false,textContent:'',
    classList:{contains:()=>false,toggle(name,value){this.hidden=value;}},
    scrollIntoView(){this.scrolled=true;},showModal(){this.open=true;},close(){this.open=false;},hidePopover(){this.dismissed=true;}});
  return elements.get(selector);
};
const utilities=[element('#view-settings'),element('#view-events')];
const tabs=['all','running','review','complete'].map(filter=>({dataset:{filter},classList:{toggle(name,value){this.selected=value;}},setAttribute(name,value){this[name]=value;}}));
let renders=0;
const navigation=vm.createContext({$:element,views:{overview:'總覽',subscriptions:'訂閱管理',jobs:'離線任務',events:'執行日誌',settings:'系統設定'},
  document:{querySelectorAll:selector=>selector==='.utility-view'?utilities:tabs},
  history:{replaceState(a,b,hash){this.hash=hash;}},tr:(parts,...values)=>parts.reduce((out,part,i)=>out+part+(values[i]??''),''),
  currentView:'overview',jobFilter:'all',jobLimit:24,render:()=>{renders++;}});
vm.runInContext(source.slice(source.indexOf('function showView('),source.indexOf('function filteredJobs()'))+source.slice(source.indexOf('function selectJobFilter('),source.indexOf('function empty(')),navigation);
vm.runInContext("showView('settings')",navigation);
assert.equal(element('#utility-dialog').open,true);assert.equal(element('#utility-title').textContent,'系統設定');
assert.equal(utilities[0].classList.hidden,false);assert.equal(utilities[1].classList.hidden,true);
vm.runInContext("showView('events')",navigation);assert.equal(element('#utility-title').textContent,'執行日誌');
vm.runInContext("showView('subscriptions')",navigation);assert.equal(element('#utility-dialog').open,false);assert.equal(element('#view-subscriptions').scrolled,true);
assert.equal(element('#profile-menu').dismissed,true);
vm.runInContext("selectJobFilter('review')",navigation);assert.equal(navigation.jobLimit,8);assert.equal(renders,1);
assert.equal(tabs[2]['aria-pressed'],'true');assert.equal(tabs[0]['aria-pressed'],'false');
console.log('Account utilities, legacy workspace links and filter pagination reset checks passed.');
