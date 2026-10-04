'use strict';
'require view';
'require fs';
'require poll';
'require ui';
'require netspeed.i18n as translations';

function T(text) { return translations.translate(text); }

function api(action, value) {
 return fs.exec('/usr/libexec/netspeed', [action].concat(value==null?[]:Array.isArray(value)?value:[value])).then(function(r) {
  if (r.code) {
   var error;try {error=JSON.parse(r.stdout).error;}catch(e){}
   throw new Error(T(error || r.stderr || '操作失败'));
  }
  return JSON.parse(r.stdout);
 });
}
function number(v) { return typeof v === 'number' && isFinite(v) && v>=0 ? v.toFixed(1) : '—'; }
function dateTime(seconds) { return new Date(seconds*1000).toLocaleString(document.documentElement.lang || 'en'); }
// Both selectors share the same controls; a native select cannot render the
// node hierarchy and would acquire a different OS-specific popup appearance.
function pickerTitle(text) { return E('summary',{'style':'cursor:pointer;padding:8px 12px;border:1px solid var(--border-color-medium,#ccd1d8);border-radius:5px;list-style:none'},text+' ▾'); }
function pickerPopup(rows) { return E('div',{'class':'netspeed-menu','style':'position:absolute;z-index:20;min-width:300px;max-width:90vw;max-height:50vh;overflow-y:auto;background:var(--background-color-high,#fff);border:1px solid #ccd1d8;padding:10px;border-radius:6px;box-shadow:0 4px 12px #0002'},rows); }
function menuChoice(text,click) { return E('button',{'type':'button','class':'netspeed-menu-item','style':'display:block;width:100%;text-align:left;margin:2px 0;padding:8px;border:0;border-radius:4px;background:transparent;color:inherit;font:inherit;cursor:pointer','click':click},text); }
function label(s) {
 if(!s)return T('自动选择');
 var cities={'10001':'合肥','10002':'南京','54':'洛杉矶','91':'洛杉矶','49':'伦敦','50':'法兰克福','51':'阿姆斯特丹','68':'新加坡','82':'东京','52':'纽约','93':'芝加哥','101':'赫尔辛基','35':'罗马','79':'布拉格','74':'波兹南','106':'贝尔格勒','104':'阿尔加拉斯蒂','10003':'城市未标明','10004':'城市未标明','10005':'巴黎','10006':'莫斯科'};
 var countries={CN:'中国',US:'美国',GB:'英国',DE:'德国',NL:'荷兰',SG:'新加坡',JP:'日本',FI:'芬兰',IT:'意大利',CZ:'捷克',PL:'波兰',RS:'塞尔维亚',GR:'希腊',BR:'巴西',AU:'澳大利亚',FR:'法国',RU:'俄罗斯'};
 var sponsors={'University of Science and Technology of China':'中国科学技术大学','Southeast University':'东南大学'};
 var cc=s.country_code || '',flag=/^[A-Z]{2}$/.test(cc)?String.fromCodePoint.apply(null,cc.split('').map(function(c){return c.charCodeAt(0)+127397;})):'🌐';
 var country=T(countries[cc] || s.country || '自定义');
 if(!countries[cc] && /^[A-Z]{2}$/.test(cc) && Intl.DisplayNames)country=new Intl.DisplayNames([document.documentElement.lang || 'en'],{type:'region'}).of(cc);
 return flag+' '+country+' · '+T(cities[String(s.id)] || s.name || s.location || '')+' · '+T(sponsors[s.sponsor] || s.sponsor || s.host || '自定义');
}
function connectionLabel(r) { return r.connection_mode==='proxy'?T('代理 · ')+(r.proxy_name || T('未知节点')):T('直连'); }
function chart(samples) {
 var max=10, total=1;
 samples.forEach(function(s) { max=Math.max(max,s.mbps || 0);total=Math.max(total,s.elapsed || 0); });
 // Rounded 1/2/5 steps adapt to device bandwidth without fractional ticks.
 function stepFor(value, floor) {
  var rough=value/4, power=Math.pow(10,Math.floor(Math.log10(Math.max(rough,floor))));
  return Math.max(floor,[1,2,5,10].map(function(n){return n*power;}).find(function(n){return n>=rough;}));
 }
 var ys=stepFor(max,10), xs=stepFor(total,1);
 max=Math.ceil(max/ys)*ys; total=Math.ceil(total/xs)*xs;
 var svg=document.createElementNS('http://www.w3.org/2000/svg','svg');
 svg.setAttribute('viewBox','0 0 720 260'); svg.setAttribute('role','img'); svg.setAttribute('aria-label',T('下载与上传速度曲线'));
 svg.style.width='100%'; svg.style.maxHeight='320px';
 function node(tag,attrs,text) {
  var n=document.createElementNS(svg.namespaceURI,tag);
  Object.keys(attrs).forEach(function(k){n.setAttribute(k,attrs[k]);});
  if(text) n.textContent=text; svg.appendChild(n); return n;
 }
 var left=75,right=705,top=20,bottom=190;
 for(var v=0;v<=max;v+=ys) {
  var y=bottom-(bottom-top)*v/max;
  node('line',{x1:left,y1:y,x2:right,y2:y,stroke:'#d7dce3','stroke-width':1});
  node('text',{x:left-10,y:y+4,fill:'#77818e','font-size':11,'text-anchor':'end'},String(v));
 }
 for(var t=0;t<=total;t+=xs)node('text',{x:left+(right-left)*t/total,y:210,fill:'#77818e','font-size':11,'text-anchor':'middle'},String(t));
 node('line',{x1:left,y1:top,x2:left,y2:bottom,stroke:'#a1aab5'});
 node('line',{x1:left,y1:bottom,x2:right,y2:bottom,stroke:'#a1aab5'});
 ['download','upload'].forEach(function(phase) {
  var rows=samples.filter(function(s){return s.phase===phase;});
  if(!rows.length)return;
  // First sample is the earliest observation, not a measured zero speed.
  // Extend it to t=0 rather than inventing a zero-valued measurement.
  var points=left+','+(bottom-(bottom-top)*(rows[0].mbps || 0)/max)+' '+rows.map(function(s,i){return (left+(right-left)*(s.elapsed || (i+1)/2)/total)+','+(bottom-(bottom-top)*(s.mbps || 0)/max);}).join(' ');
  node('polyline',{points:points,fill:'none',stroke:phase==='download'?'#16a6a1':'#6877db','stroke-width':3,'stroke-linejoin':'round'});
 });
 node('text',{x:(left+right)/2,y:233,fill:'#77818e','font-size':12,'text-anchor':'middle'},T('时间（s）'));
 node('text',{x:16,y:(top+bottom)/2,fill:'#77818e','font-size':12,'text-anchor':'middle',transform:'rotate(-90 16 '+((top+bottom)/2)+')'},T('速度（Mbps）'));
 node('text',{x:535,y:255,fill:'#16a6a1','font-size':12},T('● 下载'));
 node('text',{x:620,y:255,fill:'#6877db','font-size':12},T('● 上传'));
 return svg;
}
return view.extend({
 load:function(){return Promise.all([api('servers').catch(function(){return {};}),api('routes').catch(function(){return {available:false,nodes:[]};})]).then(function(rows){return api('status').then(function(s){s.routes=rows[1];return s;});});},
 render:function(data){
  var self=this;
  this.selection=E('select',{'style':'display:none'},[E('option',{value:'auto'},T('自动选择节点'))]);
  this.nodeTitle=pickerTitle(T('自动选择节点'));
  this.reachableNodes=E('div');this.otherNodes=E('div');
  this.nodePicker=E('details',{'style':'position:relative;min-width:260px'},[
   this.nodeTitle,pickerPopup([
    menuChoice(T('自动选择节点'),function(){self.selection.value='auto';self.nodeTitle.textContent=T('自动选择节点')+' ▾';self.nodePicker.open=false;}),
    E('details',{},[E('summary',{'style':'cursor:pointer;padding:8px'},T('可直连节点')),this.reachableNodes]),
    E('details',{},[E('summary',{'style':'cursor:pointer;padding:8px'},T('可能无法直连的节点')),this.otherNodes]),
    menuChoice(T('添加自定义节点'),function(){self.nodePicker.open=false;self.addCustom();})
   ])
  ]);
  this.connection=E('select',{'style':'display:none'});
  this.connectionTitle=pickerTitle(T('直连'));
  this.connectionNodes=E('div');
  this.connectionPicker=E('details',{'style':'position:relative;min-width:260px'},[this.connectionTitle,pickerPopup([this.connectionNodes])]);
  this.updateConnections(data.routes || {nodes:[]});
  this.connectionPicker.addEventListener('toggle',function(){if(self.connectionPicker.open)self.nodePicker.open=false;});
  this.nodePicker.addEventListener('toggle',function(){if(self.nodePicker.open)self.connectionPicker.open=false;});
  this.serverText=E('p',{'style':'color:var(--text-color-secondary,#777)'},'');
  this.message=E('span',{'role':'status','aria-live':'polite'},T('准备就绪'));
  this.startButton=E('button',{'class':'cbi-button cbi-button-action important','click':function(){var direct=self.connection.value==='direct';self.action('start',[self.selection.value,direct?'direct':'proxy',direct?'':self.connection.value]);}},T('开始测速'));
  this.stopButton=E('button',{'class':'cbi-button cbi-button-negative','style':'display:none','click':function(){self.action('stop');}},T('停止'));
  this.refreshButton=E('button',{'class':'cbi-button','style':'margin-left:8px','click':function(){Promise.all([api('servers','refresh'),api('routes')]).then(function(rows){var r=rows[0];if(r.error)throw new Error(r.error);self.updateConnections(rows[1]);self.message.textContent=r.pending?T('正在查找节点…'):T('节点已更新');}).catch(function(e){self.message.textContent=T(e.message);});}},T('刷新节点'));
  this.metrics={};
  var cards=[['download_mbps',T('下载'),'Mbps'],['upload_mbps',T('上传'),'Mbps'],['latency_ms',T('延迟'),'ms'],['jitter_ms',T('抖动'),'ms']].map(function(m){
   self.metrics[m[0]]=E('strong',{'style':'display:block;font-size:32px;font-weight:500;line-height:1.6'},'—');
   return E('div',{'style':'flex:1;min-width:120px;padding:16px;background:var(--background-color-high,#f6f7f9);border-radius:10px'},[E('div',{},m[1]),self.metrics[m[0]],E('span',{'style':'opacity:.6'},m[2])]);
  });
  this.plot=E('div',{},chart([])); this.records=E('div');
  this.clearButton=E('button',{'class':'cbi-button','click':function(){
   ui.showModal(T('清除历史记录'),[
    E('p',{},T('清除全部测速记录？')),E('div',{'class':'right'},[
     E('button',{'class':'cbi-button','click':ui.hideModal},T('取消')),
     E('button',{'class':'cbi-button cbi-button-negative','click':function(){
      api('clear-history').then(function(r){if(r.error)throw new Error(r.error);ui.hideModal();return api('status');}).then(function(s){self.update(s);}).catch(function(e){ui.addNotification(null,E('p',{},T(e.message)));});
     }},T('清除'))
    ])
   ]);
  }},T('清除历史记录'));
  var page=E('div',{'class':'cbi-section','style':'max-width:1000px;padding:24px'},[
   E('h2',{},T('一键测速')),
   E('div',{'style':'display:flex;gap:12px;flex-wrap:wrap;align-items:center;margin:20px 0'},[this.startButton,this.stopButton,this.message]),
   E('div',{'style':'display:flex;gap:12px;flex-wrap:wrap'},cards),this.serverText,this.plot,
   E('div',{'style':'display:flex;gap:16px;align-items:end;flex-wrap:wrap;margin:12px 0 24px'},[
    E('div',{},[E('div',{'style':'margin-bottom:6px'},T('连接方式')),this.connection,this.connectionPicker]),
    E('div',{},[E('div',{'style':'margin-bottom:6px'},T('测速节点')),this.selection,this.nodePicker]),this.refreshButton]),
   E('div',{'style':'display:flex;align-items:center;justify-content:space-between;margin-top:20px'},[E('h3',{},T('测速记录')),this.clearButton]),this.records
  ]);
  this.update(data);
  poll.add(function(){return api('status').then(function(s){self.update(s);}).catch(function(e){self.message.textContent=T('连接中断：')+T(e.message);});},1);
  return page;
 },
 updateConnections:function(routes){
  var self=this,selected=this.connection.value || 'direct',nodes=[{name:T('直连'),value:'direct'}].concat((routes.nodes || []).map(function(n){return {name:n.name,value:n.name};}));
  this.connection.replaceChildren.apply(this.connection,nodes.map(function(n){return E('option',{value:n.value},n.name);}));
  this.connection.value=nodes.some(function(n){return n.value===selected;})?selected:'direct';
  this.connectionTitle.textContent=nodes.find(function(n){return n.value===self.connection.value;}).name+' ▾';
  this.connectionNodes.replaceChildren.apply(this.connectionNodes,nodes.map(function(n){return menuChoice(n.name,function(){self.connection.value=n.value;self.connectionTitle.textContent=n.name+' ▾';self.connectionPicker.open=false;});}));
 },
 action:function(cmd,value){
  var self=this;
  this.startButton.disabled=true;
  api(cmd,value).then(function(r){if(r.error)throw new Error(r.error); self.message.textContent=cmd==='stop'?T('正在停止…'):T('正在准备…'); return api('status');}).then(function(s){self.update(s);}).catch(function(e){self.message.textContent=T(e.message); self.startButton.disabled=false;});
 },
 addCustom:function(){
  var self=this,name=E('input',{'class':'cbi-input-text',placeholder:T('例如：我的测速服务器')}),base=E('input',{'class':'cbi-input-text',placeholder:'https://speed.example.com/'});
  var country=E('input',{'class':'cbi-input-text',placeholder:'CN / US / JP',maxlength:2});
  var dl=E('input',{'class':'cbi-input-text',value:'backend/garbage.php'}),ul=E('input',{'class':'cbi-input-text',value:'backend/empty.php'}),ping=E('input',{'class':'cbi-input-text',value:'backend/empty.php'});
  function field(title,input){return E('label',{'style':'display:block;margin:10px 0'},[E('div',{},title),input]);}
  var error=E('p',{'role':'alert'});
  ui.showModal(T('添加自定义节点'),[
   field(T('名称'),name),field(T('国家代码'),country),field(T('LibreSpeed 服务地址'),base),
   E('details',{},[E('summary',{'style':'cursor:pointer'},T('接口路径')),field(T('下载'),dl),field(T('上传'),ul),field(T('延迟'),ping)]),error,
   E('div',{'class':'right'},[
    E('button',{'class':'cbi-button','click':ui.hideModal},T('取消')),
    E('button',{'class':'cbi-button cbi-button-action','click':function(){
     var cc=country.value.trim().toUpperCase();if(!/^[A-Z]{2}$/.test(cc)){error.textContent=T('国家代码需填写两个大写英文字母');return;}
     api('custom-add',JSON.stringify({name:name.value.trim(),country_code:cc,base_url:base.value.trim(),download_url:dl.value.trim(),upload_url:ul.value.trim(),ping_url:ping.value.trim()})).then(function(r){if(r.error)throw new Error(r.error);self.selection.appendChild(E('option',{value:r.id},name.value));self.selection.value=r.id;self.nodeTitle.textContent=label({id:r.id,name:name.value,country_code:cc,sponsor:'自定义'})+' ▾';ui.hideModal();return api('servers','refresh');}).catch(function(e){error.textContent=T(e.message);});
    }},T('添加'))
   ])
  ]);
 },
 update:function(s){
  var events=s.events || [], values={}, samples=[], server=null, phase='';
  events.forEach(function(e){
   if(e.type==='sample'){samples.push(e);values[e.phase+'_mbps']=e.mbps;}
   if(e.type==='server')server=e.server;
   if(e.type==='phase')phase=e.phase;
   if(e.type==='ping' || e.type==='result')Object.assign(values,e);
   if(e.type==='result' && e.server)server=e.server;
  });
  if(s.status==='failed'){delete values.download_mbps;delete values.upload_mbps;samples=[];}
  Object.keys(this.metrics).forEach(function(k){this.metrics[k].textContent=number(values[k]);},this);
  var signature=s.status+JSON.stringify(events);
  if(signature!==this.eventSignature){this.plot.replaceChildren(chart(samples));this.eventSignature=signature;}
  this.serverText.textContent=server?connectionLabel(s.connection || values)+T('　测速节点：')+label(server):'';
  var busy=s.running || s.status==='starting' || s.status==='discovering';
  this.startButton.disabled=busy;this.selection.disabled=busy;this.connection.disabled=busy;this.refreshButton.disabled=busy;
  [this.nodeTitle,this.connectionTitle].forEach(function(title){title.style.pointerEvents=busy?'none':'';title.setAttribute('aria-disabled',String(busy));});
  if(busy){this.nodePicker.open=false;this.connectionPicker.open=false;}
  this.stopButton.style.display=s.running?'':'none';
  this.clearButton.disabled=busy || !(s.history || []).length;
  var states={idle:'',ready:'',starting:T('正在准备…'),discovering:T('正在查找节点…'),running:({ping:T('正在测延迟…'),download:T('正在测下载…'),upload:T('正在测上传…')})[phase] || T('正在选择节点…'),complete:'',stopped:T('已停止'),failed:T('测速未完成，请换一个节点重试')};
  this.message.textContent=states[s.status] || '';
  var errs=events.filter(function(e){return e.type==='error';});
  if(s.status==='failed' && errs.length)this.message.textContent=T(errs[errs.length-1].message);
  var serversSignature=JSON.stringify(s.servers || []);
  if(s.servers && this.serversSignature!==serversSignature){
   var selected=this.selection.value;this.serversSignature=serversSignature;
   this.selection.replaceChildren(E('option',{value:'auto'},T('自动选择节点')));
   s.servers.forEach(function(server){this.selection.appendChild(E('option',{value:String(server.id)},label(server)));},this);
   this.selection.value=s.servers.some(function(server){return String(server.id)===selected;})?selected:'auto';
   var self=this;
   function choice(text,id){return menuChoice(text,function(){self.selection.value=id;self.nodeTitle.textContent=text+' ▾';self.nodePicker.open=false;});}
   this.reachableNodes.replaceChildren();
   this.otherNodes.replaceChildren(E('p',{'style':'font-size:12px;opacity:.7'},T('以下节点当前可能无法直连，可通过代理测试。')));
   s.servers.forEach(function(server){(server.reachable?self.reachableNodes:self.otherNodes).appendChild(choice(label(server),String(server.id)));});
   this.nodeTitle.textContent=(this.selection.value==='auto'?T('自动选择节点'):label(s.servers.find(function(server){return String(server.id)===self.selection.value;})))+' ▾';
  }
  var rows=(s.history || []).map(function(r){return E('tr',{},[E('td',{},dateTime(r.saved_at)),E('td',{},connectionLabel(r)),E('td',{},label(r.server)),E('td',{},number(r.download_mbps)),E('td',{},number(r.upload_mbps)),E('td',{},number(r.latency_ms)),E('td',{},number(r.jitter_ms)),E('td',{},E('button',{'class':'cbi-button','click':function(){
   ui.showModal(T('测速记录 · ')+dateTime(r.saved_at),[
    E('p',{},connectionLabel(r)+T('　测速节点：')+label(r.server)),E('p',{},T('下载 ')+number(r.download_mbps)+T(' Mbps　上传 ')+number(r.upload_mbps)+T(' Mbps　延迟 ')+number(r.latency_ms)+T(' ms　抖动 ')+number(r.jitter_ms)+' ms'),chart(r.samples || []),
    E('div',{'class':'right'},E('button',{'class':'cbi-button','click':ui.hideModal},T('关闭')))
   ]);
  }},T('查看记录')))]);},this);
  this.records.replaceChildren(rows.length?E('div',{'style':'overflow-x:auto'},E('table',{'class':'table'},[E('tr',{'class':'tr table-titles'},[T('时间'),T('连接方式'),T('测速节点'),T('下载 Mbps'),T('上传 Mbps'),T('延迟 ms'),T('抖动 ms'),''].map(function(t){return E('th',{},t);} ))].concat(rows))):E('p',{'style':'opacity:.6'},T('完成一次测速后，结果会保存在这里。')));
 },
 handleSaveApply:null,handleSave:null,handleReset:null
});
