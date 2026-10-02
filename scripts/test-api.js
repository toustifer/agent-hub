const protocol = process.env.HUB_PROTOCOL || 'https';
const httpModule = protocol === 'http' ? require('http') : require('https');
const hostname = process.env.HUB_HOST || 'hub.stifer.xyz';
const port = process.env.HUB_PORT ? parseInt(process.env.HUB_PORT, 10) : (protocol === 'http' ? 80 : 443);
const API_KEY = process.env.HUB_API_KEY;
const BUSINESS = process.env.HUB_BUSINESS_CODE || process.env.HUB_BUSINESS;
if (!API_KEY || !BUSINESS) {
  console.error('Set HUB_API_KEY and HUB_BUSINESS_CODE. Magic defaults (agent-company-worker / ai-medbox) were removed.');
  process.exit(1);
}
function api(method, path, data, headers) {
  return new Promise((resolve) => {
    const opts = {hostname, port, path, method, headers:{'Content-Type':'application/json','X-API-Key':API_KEY,'X-Business-Code':BUSINESS,...headers}};
    const req = httpModule.request(opts, res => {let d='';res.on('data',c=>d+=c);res.on('end',()=>{try{resolve({code:res.statusCode,data:JSON.parse(d)})}catch(e){resolve({code:res.statusCode,data:d})}})});
    if(data) req.write(JSON.stringify(data));
    req.end();
  });
}
async function main() {
  let pass=0, fail=0;
  function check(name, ok, detail) { if(ok){pass++;console.log('PASS '+name+(detail?' - '+detail:''))}else{fail++;console.log('FAIL '+name+(detail?' - '+detail:''))} }

  // 1. Lock acquire
  const lock = await api('POST','/v1/hub/locks/acquire',{resource_key:'test.lock'+Date.now(),worker_id:'test',ttl_seconds:30});
  const hasToken = !!(lock.data?.data?.holder_token);
  check('Lock acquire', hasToken);
  const holderToken = lock.data?.data?.holder_token;
  if(!holderToken) { console.log('ABORT: no token'); return; }

  // 2. Lock renew (valid)
  const renew = await api('POST','/v1/hub/locks/renew',{holder_token:holderToken,ttl_seconds:30});
  check('Lock renew (valid token)', renew.code===200);

  // 3. Renew bad token (should be rejected)
  const bad = await api('POST','/v1/hub/locks/renew',{holder_token:'bad-'+Date.now(),ttl_seconds:30});
  check('Renew bad token rejected', bad.code===500, bad.data?.message);

  // 4. Lock release
  const rel = await api('POST','/v1/hub/locks/release',{holder_token:holderToken});
  check('Lock release', rel.code===200);

  // 5. Playbook dedup: same title = same ID
  const sid = 'test-dedup-'+Date.now();
  const p1 = await api('POST','/v1/hub/playbooks',{category:'patterns',title:sid,content:'v1',tags:[],worker_id:'test'});
  const p2 = await api('POST','/v1/hub/playbooks',{category:'patterns',title:sid,content:'v2-updated',tags:[],worker_id:'test'});
  const id1 = p1.data?.data?.id, id2 = p2.data?.data?.id;
  check('Playbook upsert (same ID)', id1 && id1===id2, 'id1='+id1+' id2='+id2);
  check('Playbook content updated', p2.data?.data?.content==='v2-updated', p2.data?.data?.content);

  // 6. Playbook search with API key
  const srch = await api('GET','/v1/hub/playbooks/search?q=&limit=2',null);
  check('Search with API key', srch.code===200 && (srch.data?.data||[]).length>0);

  // 7. Search without auth (should be rejected now)
  const badSrch = await api('GET','/v1/hub/playbooks/search?q=&limit=1',null,{'X-API-Key':'','X-Business-Code':''});
  check('Search without auth rejected', badSrch.code===401);

  console.log('\n=== '+pass+' passed, '+fail+' failed ===');
}
main();
