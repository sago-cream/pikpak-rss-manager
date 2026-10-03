'use strict';
const themeMedia=window.matchMedia('(prefers-color-scheme: dark)');
let themePreference;
try{themePreference=localStorage.getItem('theme');}catch{}
function applyTheme(){
  const dark=themePreference==='dark'||(themePreference!=='light'&&themeMedia.matches);
  document.documentElement.dataset.theme=dark?'dark':'light';
  for(const button of document.querySelectorAll('.theme-toggle')){
    const label=dark?t('切換淺色模式'):t('切換深色模式');
    button.textContent=dark?'☀':'☾';button.title=label;button.setAttribute('aria-label',label);button.setAttribute('aria-pressed',String(dark));
  }
}
// This script runs before styles load to apply the saved/system preference
// without flashing a light page. Controls are localized after DOM initialization.
applyTheme();
document.addEventListener('DOMContentLoaded',applyTheme);
document.addEventListener('click',event=>{
  if(!event.target.closest('.theme-toggle'))return;
  themePreference=document.documentElement.dataset.theme==='dark'?'light':'dark';
  try{localStorage.setItem('theme',themePreference);}catch{}
  applyTheme();
});
themeMedia.addEventListener('change',applyTheme);
window.addEventListener('storage',event=>{if(event.key==='theme'||event.key===null){themePreference=event.newValue;applyTheme();}});
