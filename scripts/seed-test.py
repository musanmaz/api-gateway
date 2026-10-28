#!/usr/bin/env python3
"""Register the two explicit test hostnames through the authenticated API.
Existing matching services are reused. An unexpected definition is never overwritten.
"""
import json
import time
import urllib.request
from pathlib import Path

root = Path(__file__).resolve().parents[1]
token = (root / '.local/operator.token').read_text().strip()
base = 'http://127.0.0.1:9080'
def request(path, payload=None, key=None):
    headers={'Authorization':'Bearer '+token,'Content-Type':'application/json'}
    if key: headers['Idempotency-Key']=key
    req=urllib.request.Request(base+path, data=json.dumps(payload).encode() if payload is not None else None,headers=headers)
    with urllib.request.urlopen(req,timeout=15) as response: return json.load(response)
boot=request('/api/v1/bootstrap')
for host,name,backend in [('api1.mehmet.tech','catalog-api','backend-a'),('api2.mehmet.tech','orders-api','backend-b')]:
    existing=request('/api/v1/services?q='+host)['items']
    service=next((s for s in existing if host in s['hostnames']),None)
    if service:
        endpoints=service['routes'][0]['upstream']['endpoints'] if service['routes'] else []
        if not endpoints or endpoints[0]['address']!=backend or service['routes'][0]['path']!='/':
            raise SystemExit('Refusing to overwrite an existing route for '+host)
    else:
        service=request('/api/v1/services',{
          'name':name,'owner':'platform','importance':'low','description':'Public HTTPS gateway test service',
          'environmentId':boot['environmentId'],'teamId':boot['teamId'],'fleetId':boot['fleetId'],
          'listenerId':boot['listener']['id'],'hostnames':[host],
          'routes':[{'path':'/','matchType':'prefix','methods':['GET','HEAD'], 'upstream':{
            'name':backend,'algorithm':'round_robin','connectTimeoutMs':1000,'requestTimeoutMs':5000,
            'endpoints':[{'address':backend,'port':8080,'weight':1}]}}]
        })
    check=request('/api/v1/services/'+service['id']+'/validate',{})
    if not check['valid']: raise SystemExit(str(check['errors']))
    # Do not create another revision for an unchanged, already published service.
    if service['status']!='active' or service['version']!=service['publishedVersion']:
        result=request('/api/v1/services/'+service['id']+'/publish',{'expectedVersion':service['version']},key='test-seed-'+host+'-'+str(service['version']))
        print(host+' published as revision '+str(result['revision']['id']))
    else: print(host+' already published')
for attempt in range(30):
    view=request('/api/v1/overview')
    if len(view['nodes'])==2 and all(n['heartbeat']=='fresh' and n['desiredRevision'] is not None and n['ackedRevision']==n['desiredRevision'] for n in view['nodes']):
        print('Both gateway nodes acknowledged the current revision.')
        break
    time.sleep(1)
else: raise SystemExit('Gateway nodes did not converge in 30 seconds')
