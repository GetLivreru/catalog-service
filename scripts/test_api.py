#!/usr/bin/env python3
"""Exercise a running catalog API; create isolated records and clean them up."""
from datetime import datetime
import json
import os
from pathlib import Path
import urllib.error
import urllib.request
import uuid

base = os.getenv('CATALOG_URL', 'http://localhost:8080')
records = []
created = []
suffix = uuid.uuid4().hex[:10]


def request(label, method, path, expected, body=None):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(base + '/v1' + path, data=data, method=method,
                                 headers={'Content-Type': 'application/json'})
    try:
        response = urllib.request.urlopen(req, timeout=10)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        status = response.status
        raw = response.read().decode()
        result = json.loads(raw) if raw else None
    records.append(dict(label=label, method=method, path='/v1' + path,
                        request=body, status=status, response=result))
    assert status == expected, (label, expected, status, result)
    if expected == 201:
        assert 'updated_at' not in result and 'id' not in result
        created.append((path.split('/')[1], result['guid']))
    if expected >= 400:
        assert isinstance(result.get('error'), str)
    print(f'PASS {status} {label}')
    return result


try:
    cat = request('Create category', 'POST', '/category/create', 201, {'name': 'API category ' + suffix})
    cid = cat['guid']
    request('Duplicate category', 'POST', '/category/create', 409, {'name': cat['name']})
    product = dict(name='API product ' + suffix, description='API smoke test', price=12345, category_guid=cid)
    prod = request('Create product', 'POST', '/product/create', 201, product)
    pid = prod['guid']
    request('Duplicate product', 'POST', '/product/create', 409, product)
    cats = request('List categories', 'POST', '/category/list', 200)
    assert any(item['guid'] == cid for item in cats['data'])
    items = request('List products', 'POST', '/product/list', 200, {})
    assert any(item['guid'] == pid for item in items['data'])
    items = request('Filter products', 'POST', '/product/list', 200, {'category_guid': cid})
    assert len(items['data']) == 1 and items['data'][0]['guid'] == pid
    items = request('Empty product list is an array', 'POST', '/product/list', 200, {'category_guid': str(uuid.uuid4())})
    assert items['data'] == []
    request('Category with linked products', 'DELETE', '/category/' + cid, 409)
    request('Product with missing category', 'POST', '/product/create', 404,
            dict(product, name='Missing category ' + suffix, category_guid=str(uuid.uuid4())))
    patched = request('PATCH name without price', 'PATCH', '/product/' + pid, 200, {'name': 'Renamed ' + suffix})
    assert patched['price'] == product['price'] and patched['description'] == product['description']
    assert abs((datetime.fromisoformat(patched['created_at']) - datetime.fromisoformat(prod['created_at'])).total_seconds()) < 0.000002
    patched = request('PATCH price zero preserves price', 'PATCH', '/product/' + pid, 200, {'price': 0})
    assert patched['price'] == product['price']
    request('Reject negative PATCH price', 'PATCH', '/product/' + pid, 400, {'price': -1})
    request('Reject missing create fields', 'POST', '/product/create', 400, {'name': 'Invalid'})
    request('Reject empty category name', 'POST', '/category/create', 400, {'name': ''})
    request('Reject invalid GUID', 'DELETE', '/product/not-a-guid', 400)
    request('Reject invalid category filter', 'POST', '/product/list', 400, {'category_guid': 'bad'})
    for kind in ('category', 'product'):
        missing = '/' + kind + '/' + str(uuid.uuid4())
        request('Missing ' + kind + ' update', 'PATCH', missing, 404, {})
        request('Missing ' + kind + ' delete', 'DELETE', missing, 404)
    cat2 = request('Create second category', 'POST', '/category/create', 201, {'name': 'Second ' + suffix})
    request('Reject duplicate category rename', 'PATCH', '/category/' + cat2['guid'], 409, {'name': cat['name']})
    request('Allow unchanged category name', 'PATCH', '/category/' + cid, 200, {'name': cat['name']})
    request('Rename category', 'PATCH', '/category/' + cid, 200, {'name': 'Renamed category ' + suffix})
    prod2 = request('Create second product', 'POST', '/product/create', 201, dict(product, name='Second product ' + suffix))
    request('Reject duplicate product rename', 'PATCH', '/product/' + prod2['guid'], 409, {'name': 'Renamed ' + suffix})
    request('Reject missing category on PATCH', 'PATCH', '/product/' + pid, 404, {'category_guid': str(uuid.uuid4())})
    patched = request('Update price description and category', 'PATCH', '/product/' + pid, 200,
                      {'price': 23456, 'description': '', 'category_guid': cat2['guid']})
    assert patched['price'] == 23456 and patched['description'] == '' and patched['category_guid'] == cat2['guid']
finally:
    # Products must be removed before categories, irrespective of creation order.
    for kind, guid in sorted(created, key=lambda item: item[0] != 'product'):
        result = request('Delete test ' + kind, 'DELETE', '/' + kind + '/' + guid, 200)
        assert result is None
    output = Path(os.getenv('API_REPORT', '/tmp/task-1005-api-results.json'))
    output.write_text(json.dumps(records, indent=2, ensure_ascii=False) + '\n')
    print('Report:', output)
