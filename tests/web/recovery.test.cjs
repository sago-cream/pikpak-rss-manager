'use strict';
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const source=fs.readFileSync('internal/web/static/app.js','utf8');
const handler=source.slice(source.indexOf('function bindRecoveryForm(){'),source.indexOf('async function start(){'));
(async()=>{
  let submit,fail=true;
  const calls=[],hidden=new Set();
  const button={disabled:false};
  const elements={};
  for(const id of ['recovery-code','recovery-password','recovery-confirm','recovery-error','recovery-title','recovery-help','recovery-success']){
    elements['#'+id]={value:'',textContent:'',classList:{add:()=>hidden.add(id),remove:()=>hidden.delete(id)}};
  }
  const form={addEventListener:(event,fn)=>{submit=fn;},querySelector:()=>button,
    reset:()=>{for(const id of ['code','password','confirm'])elements['#recovery-'+id].value='';},
    classList:{add:()=>hidden.add('recovery-form')}};
  elements['#recovery-form']=form;
  const ctx=vm.createContext({$:key=>elements[key],t:value=>value,
    api:async(...args)=>{assert.equal(button.disabled,true);calls.push(args);if(fail)throw Error('Expired fixture code');return {ok:true};}});
  vm.runInContext(handler+'\nbindRecoveryForm();',ctx);
  function fill(confirm='new'){
    elements['#recovery-code'].value='  fixture-code  ';
    elements['#recovery-password'].value='new';
    elements['#recovery-confirm'].value=confirm;
  }
  fill('different');await submit({preventDefault(){}});
  assert.equal(calls.length,0);
  assert.equal(elements['#recovery-error'].textContent,'兩次輸入的密碼不一致');
  fill();await submit({preventDefault(){}});
  assert.equal(elements['#recovery-error'].textContent,'Expired fixture code');
  assert.equal(hidden.has('recovery-form'),false);
  assert.equal(button.disabled,false);
  assert.equal(elements['#recovery-code'].value,'');
  fail=false;fill();await submit({preventDefault(){}});
  assert.equal(calls.length,2);
  assert.equal(calls[1][0],'/api/password/recovery');
  assert.equal(calls[1][1],'POST');
  assert.equal(calls[1][2].code,'fixture-code');
  assert.equal(calls[1][2].new_password,'new');
  assert.equal(elements['#recovery-error'].textContent,'');
  assert.equal(elements['#recovery-password'].value,'');
  assert.equal(elements['#recovery-title'].textContent,'密碼已重設');
  assert.equal(hidden.has('recovery-form'),true);
  assert.equal(hidden.has('recovery-help'),true);
  assert.equal(hidden.has('recovery-success'),false);
  assert.equal(button.disabled,false);
  console.log('Recovery mismatch, failure, retry and successful sign-in handoff checks passed.');
})().catch(error=>{console.error(error);process.exitCode=1;});
