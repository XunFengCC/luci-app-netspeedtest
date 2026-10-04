'use strict';
'require baseclass';

// LuCI renders its resolved system language on <html>. Keeping the catalog
// beside this standalone view lets direct installation and package builds use
// identical translations without requiring a host-side lmo compiler.
var rows = [
 ['测速隔离用户编号已被其他服务使用','測速隔離使用者編號已被其他服務使用','The isolated test UID is already used by another service'],
 ['保存自定义节点失败','儲存自訂節點失敗','Failed to save the custom server'],
 ['俄罗斯','俄羅斯','Russia'],['莫斯科','莫斯科','Moscow'],
 ['OpenClash节点不可用，请刷新后重试','OpenClash 節點不可用，請重新整理後重試','OpenClash node unavailable; refresh and retry'],['任务正在准备，请稍后重试','任務正在準備，請稍後重試','Task is starting; retry shortly'],['接口需填写相对路径','介面需填寫相對路徑','Enter relative endpoint paths'],['无效操作','無效操作','Invalid action'],['无效节点','無效節點','Invalid server'],['无效连接方式','無效連線方式','Invalid connection mode'],['最多20个自定义节点','最多 20 個自訂節點','Up to 20 custom servers'],['测速正在进行','測速正在進行','A test is already running'],['测速结果不完整','測速結果不完整','Incomplete test result'],['清除记录失败','清除記錄失敗','Failed to clear history'],['自定义节点地址或接口路径无效','自訂節點網址或介面路徑無效','Invalid custom server URL or endpoint'],['请在测速结束后添加节点','請在測速結束後新增節點','Add servers after the test finishes'],['请在测速结束后清除记录','請在測速結束後清除記錄','Clear history after the test finishes'],['请输入名称及不含账号密码的 HTTP/HTTPS 地址','請輸入名稱及不含帳號密碼的 HTTP/HTTPS 網址','Enter a name and an HTTP/HTTPS URL without credentials'],
 ['无法启动所选测速代理','無法啟動所選測速代理','Unable to start the selected test proxy'],['无法建立测速直连路径','無法建立測速直連路徑','Unable to establish the direct test route'],['测速期间网络规则发生变化，请重新测速','測速期間網路規則發生變化，請重新測速','Network rules changed during the test; retry'],['测速未完成，请稍后重试','測速未完成，請稍後重試','Test failed; retry later'],['测速结果不完整，请重新测速','測速結果不完整，請重新測速','Incomplete result; retry'],['测速超时，请换一个节点重试','測速逾時，請換一個節點重試','Test timed out; try another server'],
 ['城市未标明','城市未標明','City not specified'],['巴黎','巴黎','Paris'],
 ['国家代码','國家代碼','Country code'],['国家代码需填写两个大写英文字母','國家代碼需填寫兩個大寫英文字母','Enter a two-letter country code'],
 ['纽约','紐約','New York'],['芝加哥','芝加哥','Chicago'],['赫尔辛基','赫爾辛基','Helsinki'],['罗马','羅馬','Rome'],['布拉格','布拉格','Prague'],['波兹南','波茲南','Poznan'],['贝尔格勒','貝爾格勒','Belgrade'],['阿尔加拉斯蒂','阿爾加拉斯蒂','Argalasti'],
 ['芬兰','芬蘭','Finland'],['意大利','義大利','Italy'],['捷克','捷克','Czech Republic'],['波兰','波蘭','Poland'],['塞尔维亚','塞爾維亞','Serbia'],['希腊','希臘','Greece'],['巴西','巴西','Brazil'],['澳大利亚','澳洲','Australia'],['法国','法國','France'],
 ['New York','紐約','New York'],['Chicago','芝加哥','Chicago'],['Helsinki','赫爾辛基','Helsinki'],['Rome','羅馬','Rome'],['Prague','布拉格','Prague'],['Poznan','波茲南','Poznan'],['Belgrade','貝爾格勒','Belgrade'],['Argalasti','阿爾加拉斯蒂','Argalasti'],['Brazil','巴西','Brazil'],
 ['一键测速','一鍵測速','Speed test'],
 ['操作失败','操作失敗','Operation failed'],
 ['自动选择','自動選擇','Automatic'],
 ['自动选择节点','自動選擇節點','Automatic server'],
 ['自动选择节点 ▾','自動選擇節點 ▾','Automatic server ▾'],
 ['中国','中國','China'],['美国','美國','United States'],['英国','英國','United Kingdom'],['德国','德國','Germany'],['荷兰','荷蘭','Netherlands'],['新加坡','新加坡','Singapore'],['日本','日本','Japan'],
 ['合肥','合肥','Hefei'],['南京','南京','Nanjing'],['洛杉矶','洛杉磯','Los Angeles'],['伦敦','倫敦','London'],['法兰克福','法蘭克福','Frankfurt'],['阿姆斯特丹','阿姆斯特丹','Amsterdam'],['东京','東京','Tokyo'],
 ['中国科学技术大学','中國科學技術大學','USTC'],['东南大学','東南大學','Southeast University'],['自定义','自訂','Custom'],
 ['代理 · ','代理 · ','Proxy · '],['直连','直連','Direct'],['未知节点','未知節點','Unknown proxy'],
 ['下载与上传速度曲线','下載與上傳速度曲線','Download and upload speed chart'],['时间（s）','時間（s）','Time (s)'],['速度（Mbps）','速度（Mbps）','Speed (Mbps)'],['● 下载','● 下載','● Download'],['● 上传','● 上傳','● Upload'],
 ['可直连节点','可直連節點','Directly reachable servers'],['可能无法直连的节点','可能無法直連的節點','Possibly unreachable servers'],['以下节点当前可能无法直连，可通过代理测试。','以下節點目前可能無法直連，可透過代理測試。','These servers may require a proxy.'],
 ['添加自定义节点','新增自訂節點','Add custom server'],['开始测速','開始測速','Start test'],['停止','停止','Stop'],['刷新节点','重新整理節點','Refresh servers'],['准备就绪','準備就緒','Ready'],['节点已更新','節點已更新','Servers refreshed'],
 ['下载','下載','Download'],['上传','上傳','Upload'],['延迟','延遲','Latency'],['抖动','抖動','Jitter'],['连接方式','連線方式','Connection'],['测速节点','測速節點','Test server'],['　测速节点：','　測速節點：',' · Server: '],
 ['清除历史记录','清除歷史記錄','Clear history'],['清除全部测速记录？','清除全部測速記錄？','Clear all test records?'],['取消','取消','Cancel'],['清除','清除','Clear'],['测速记录','測速記錄','Test history'],['测速记录 · ','測速記錄 · ','Test record · '],['查看记录','查看記錄','View'],['关闭','關閉','Close'],['时间','時間','Time'],
 ['下载 Mbps','下載 Mbps','Download Mbps'],['上传 Mbps','上傳 Mbps','Upload Mbps'],['延迟 ms','延遲 ms','Latency ms'],['抖动 ms','抖動 ms','Jitter ms'],['下载 ','下載 ','Download '],[' Mbps　上传 ',' Mbps　上傳 ',' Mbps · Upload '],[' Mbps　延迟 ',' Mbps　延遲 ',' Mbps · Latency '],[' ms　抖动 ',' ms　抖動 ',' ms · Jitter '],
 ['完成一次测速后，结果会保存在这里。','完成一次測速後，結果會儲存在這裡。','Your completed tests will appear here.'],
 ['正在停止…','正在停止…','Stopping…'],['正在准备…','正在準備…','Preparing…'],['正在查找节点…','正在尋找節點…','Finding servers…'],['正在测延迟…','正在測延遲…','Measuring latency…'],['正在测下载…','正在測下載…','Testing download…'],['正在测上传…','正在測上傳…','Testing upload…'],['正在选择节点…','正在選擇節點…','Selecting server…'],['已停止','已停止','Stopped'],['连接中断：','連線中斷：','Connection lost: '],['测速未完成，请换一个节点重试','測速未完成，請換一個節點重試','Test failed; try another server'],
 ['例如：我的测速服务器','例如：我的測速伺服器','e.g. My speed test server'],['名称','名稱','Name'],['LibreSpeed 服务地址','LibreSpeed 服務網址','LibreSpeed service URL'],['接口路径','介面路徑','Endpoint paths'],['添加','新增','Add'],
 ['测速超时，请重试或更换节点','測速逾時，請重試或更換節點','Test timed out; retry or choose another server'],['测速已取消','測速已取消','Test cancelled'],['已达到本次测速流量上限','已達到本次測速流量上限','Test data limit reached'],['测速节点无效，请重新选择','測速節點無效，請重新選擇','Invalid server; select again'],['当前没有可用的国内测速节点，请稍后重试','目前沒有可用的國內測速節點，請稍後重試','No domestic server available; retry later'],['当前没有可用的测速节点，请检查代理或稍后重试','目前沒有可用的測速節點，請檢查代理或稍後重試','No server available; check the proxy or retry later'],['节点验证未完成，请更换节点或稍后重试','節點驗證未完成，請更換節點或稍後重試','Server verification failed; choose another server or retry later'],['测速节点响应异常，请更换节点或稍后重试','測速節點回應異常，請更換節點或稍後重試','Invalid server response; choose another server or retry later'],['网络测试失败，请检查网络后重试','網路測試失敗，請檢查網路後重試','Network test failed; check your connection']
];
var catalog = {};
rows.forEach(function(row) { catalog[row[0]]=row; });
return baseclass.extend({
 translate:function(text) {
  var lang=(document.documentElement.lang || 'en').toLowerCase().replace(/_/g,'-');
  var index=/^zh-(tw|hk|hant)/.test(lang)?1:/^zh/.test(lang)?0:2;
  return catalog[text]?catalog[text][index]:text;
 }
});
