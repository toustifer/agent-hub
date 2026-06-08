// cross-device-test.js — 模拟两台机器同时协作
// 场景: Windows(worker-win) 和 Mac(worker-mac) 编辑同一项目的不同文件

const https = require('https');
const HUB = 'hub.stifer.xyz';
const WH = {'X-API-Key':'agent-company-worker','X-Business-Code':'ai-medbox'};

function api(method, path, data, headers) {
  return new Promise((resolve) => {
    const opts = {hostname:HUB,path,method,headers:{'Content-Type':'application/json',...headers}};
    const req = https.request(opts, res => {let d='';res.on('data',c=>d+=c);res.on('end',()=>{try{resolve({code:res.statusCode,data:JSON.parse(d)})}catch(e){resolve({code:res.statusCode,data:d})}})});
    if(data) req.write(JSON.stringify(data));
    req.end();
  });
}

async function main() {
  let pass=0, fail=0;
  function check(name, ok, detail) { if(ok){pass++;console.log('PASS '+name+(detail?' - '+detail:''))}else{fail++;console.log('FAIL '+name+(detail?' - '+detail:''))} }

  console.log('=== 场景1: 心跳上报 ===');
  const hbWin = await api('POST','/v1/hub/workers/heartbeat',{worker_id:'worker-win-test',version:'1.0',host:'windows-dev',pid:12345},WH);
  check('Windows 心跳', hbWin.code===200);
  const hbMac = await api('POST','/v1/hub/workers/heartbeat',{worker_id:'worker-mac-test',version:'1.0',host:'mac-m1',pid:67890},WH);
  check('Mac 心跳', hbMac.code===200);

  // 验证两台机器的 worker 都在列表中
  const workers = await api('GET','/v1/hub/workers?business=ai-medbox',null,WH);
  const winInList = (workers.data?.data||[]).some(w=>w.worker_id==='worker-win-test');
  const macInList = (workers.data?.data||[]).some(w=>w.worker_id==='worker-mac-test');
  check('Windows worker 可见', winInList);
  check('Mac worker 可见', macInList);

  console.log('\n=== 场景2: 文件锁互斥 ===');
  // Windows 锁住文件A
  const lockWin = await api('POST','/v1/hub/locks/acquire',{resource_key:'ai-medbox.pages/Homepage/Homepage.js',worker_id:'worker-win-test',ttl_seconds:60},WH);
  const winLocked = !!(lockWin.data?.data?.holder_token);
  check('Windows 锁定文件A', winLocked);
  const winToken = lockWin.data?.data?.holder_token;

  // Mac 尝试锁同一个文件 — 应该失败（409 或 500）
  const lockMac = await api('POST','/v1/hub/locks/acquire',{resource_key:'ai-medbox.pages/Homepage/Homepage.js',worker_id:'worker-mac-test',ttl_seconds:60},WH);
  check('Mac 无法锁定同一文件', lockMac.code!==200, 'code='+lockMac.code+' (expected conflict)');

  // Mac 锁另一个文件 — 应该成功
  const lockMacB = await api('POST','/v1/hub/locks/acquire',{resource_key:'ai-medbox.pages/Manage/Manage.js',worker_id:'worker-mac-test',ttl_seconds:60},WH);
  check('Mac 锁定文件B成功', !!(lockMacB.data?.data?.holder_token));
  const macToken = lockMacB.data?.data?.holder_token;

  // 验证锁列表显示两个持有者
  const locks = await api('GET','/v1/hub/locks?business=ai-medbox',null,WH);
  const locksList = locks.data?.data||[];
  check('锁列表有两把锁', locksList.length>=2, locksList.length+' locks');

  // Windows 释放锁
  await api('POST','/v1/hub/locks/release',{holder_token:winToken},WH);
  // 现在 Mac 应该能锁住文件A
  const lockMacA2 = await api('POST','/v1/hub/locks/acquire',{resource_key:'ai-medbox.pages/Homepage/Homepage.js',worker_id:'worker-mac-test',ttl_seconds:60},WH);
  check('Windows释放后Mac可锁定', !!(lockMacA2.data?.data?.holder_token));
  await api('POST','/v1/hub/locks/release',{holder_token:macToken},WH);
  await api('POST','/v1/hub/locks/release',{holder_token:lockMacA2.data?.data?.holder_token},WH);

  console.log('\n=== 场景3: 事件记录与可见性 ===');
  await api('POST','/v1/hub/events',{actor:'worker-win-test',event_type:'task.started',payload:{task_id:'T99',title:'跨设备测试'}},WH);
  await api('POST','/v1/hub/events',{actor:'worker-mac-test',event_type:'task.started',payload:{task_id:'T100',title:'另一台机器的任务'}},WH);

  const events = await api('GET','/v1/hub/events?business=ai-medbox&limit=5',null,WH);
  const evList = events.data?.data||[];
  const winEvent = evList.find(e=>e.actor==='worker-win-test');
  const macEvent = evList.find(e=>e.actor==='worker-mac-test');
  check('Windows 事件可见', !!winEvent);
  check('Mac 事件可见', !!macEvent);

  console.log('\n=== 场景4: Playbook 共享 ===');
  const pbTitle = '跨设备测试-playbook-'+Date.now();
  // Windows 写入一条经验
  const pb = await api('POST','/v1/hub/playbooks',{category:'patterns',title:pbTitle,content:'跨设备测试：Windows机器发现的最佳实践',tags:['test','cross-device'],worker_id:'worker-win-test'},WH);
  check('Windows 创建 playbook', pb.code===200);

  // Mac 搜索这条经验
  const search = await api('GET','/v1/hub/playbooks/search?q='+encodeURIComponent('跨设备')+'&limit=5',null,WH);
  const found = (search.data?.data||[]).some(p=>p.title===pbTitle);
  check('Mac 能搜到 Windows 的经验', found);

  console.log('\n=== 场景5: 续期失败检测 ===');
  const badRenew = await api('POST','/v1/hub/locks/renew',{holder_token:'expired-token-xyz',ttl_seconds:60},WH);
  check('过期锁续期被拒绝', badRenew.code===500, badRenew.data?.message);

  console.log('\n========================================');
  console.log('  跨设备测试: '+pass+'/'+(pass+fail)+' 通过');
  console.log('========================================');
}
main();
