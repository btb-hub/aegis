"""Idempotently seed the isolated sandbox through authenticated Aegis APIs."""
import http.cookiejar
import json
import time
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import HTTPCookieProcessor, HTTPRedirectHandler, Request, build_opener

HOST = 'http://aegis:3000'
WORKSPACE = '00000000-0000-0000-0000-000000000001'
PREFIX = '/modules/anomaly-detector/api/v1/'
SOURCE = '20000000-0000-4000-8000-000000000001'
METRIC = '20000000-0000-4000-8000-000000000002'
DEADLINE = time.monotonic() + 180
REGISTRY_PATH = Path('/state/resources.json')
REGISTRY = json.loads(REGISTRY_PATH.read_text()) if REGISTRY_PATH.exists() else {}


def timeout():
    return max(.01, min(10, DEADLINE - time.monotonic()))


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


client = build_opener(HTTPCookieProcessor(http.cookiejar.CookieJar()), NoRedirect())


def api(path, body=None):
    url = HOST + path
    if path.startswith(PREFIX):
        url += ('&' if '?' in url else '?') + urlencode({'workspace_id': WORKSPACE})
    request = Request(url, None if body is None else json.dumps(body).encode(),
                      {'Content-Type': 'application/json', 'Origin': 'http://127.0.0.1:18080', 'Host': '127.0.0.1:18080'})
    try:
        with client.open(request, timeout=timeout()) as response:
            return json.load(response)
    except HTTPError as error:
        print(f'{path}: HTTP {error.code}: {error.read(512).decode(errors="replace")}', flush=True)
        raise


def wait_for(label, fn):
    while time.monotonic() < DEADLINE:
        try:
            return fn()
        except (HTTPError, URLError, TimeoutError, ValueError, AssertionError):
            time.sleep(max(0, min(2, DEADLINE - time.monotonic())))
    raise RuntimeError(f'{label} did not become ready; inspect docker compose logs')


def login():
    try:
        client.open(Request(HOST + '/auth/dev/login?role=admin', headers={'Host': '127.0.0.1:18080'}), timeout=timeout())
    except HTTPError as error:
        if error.code != 302:
            raise


def grant_ready():
    status = api('/api/v1/modules/anomaly-detector/capabilities')
    if not status.get('enabled'):
        return False
    assert status.get('available') is True or status.get('healthy') is True, status
    # The module independently refreshes its host grant, so also poll its service capabilities.
    request = Request('http://anomaly-detector:8080/api/v1/capabilities',
                      headers={'Authorization': 'Bearer local-sandbox-service-9876543210-fedcba'})
    with client.open(request, timeout=timeout()) as response:
        assert json.load(response).get('entitled') is True
    return True


def samples_ready():
    request = Request('http://mimir:9009/prometheus/api/v1/query?' + urlencode({'query': 'anomaly_demo_value'}),
                      headers={'X-Scope-OrgID': 'local-sandbox'})
    with client.open(request, timeout=timeout()) as response:
        assert len(json.load(response)['data']['result']) == 2


def ensure(kind, item):
    existing = api(PREFIX + kind)
    key = kind + '/' + item['name']
    for value in existing:
        if value['id'] == REGISTRY.get(key) or (key not in REGISTRY and value['name'] == item['name']):
            remember(key, value['id'])
            return value
    value = api(PREFIX + kind, item)
    remember(key, value['id'])
    return value


def remember(key, value):
    REGISTRY[key] = value
    REGISTRY_PATH.parent.mkdir(parents=True, exist_ok=True)
    temporary = REGISTRY_PATH.with_suffix('.tmp')
    temporary.write_text(json.dumps(REGISTRY))
    temporary.replace(REGISTRY_PATH)


def main():
    wait_for('dev login', login)
    if not wait_for('module grant', grant_ready):
        print('Module access is paused in Aegis Settings; retained resources were left untouched.', flush=True)
        return
    wait_for('Mimir samples', samples_ready)
    source = ensure('sources', {'id': SOURCE, 'name': 'Local Mimir', 'kind': 'mimir', 'url': 'http://mimir:9009',
                      'api_prefix': '/prometheus', 'tenant': 'local-sandbox', 'enabled': True,
                      'suppress_on_failure': True, 'max_series': 10, 'max_points': 50000, 'timeout_seconds': 10})
    controls = {'step_seconds': 10, 'cache_overlap_seconds': 60, 'ignore_last_seconds': 5,
                'median_timezone': 'UTC', 'poll_seconds': 10, 'observation_days': 1,
                'burst_count': 1, 'burst_window': 1, 'cooldown_seconds': 0, 'recovery_seconds': 15,
                'freshness_seconds': 90, 'retention_days': 7, 'max_snapshots': 100}
    algorithm = {'method': 'interval', 'min': 0, 'max': 20, 'period': 6, 'only_last': True,
                 'direction': 'both', 'alpha': .01, 'max_anoms': .1, 'threshold': 'none',
                 'order': 5, 'ar_threshold': 3, 'stable': True, 'sensitivity': 1,
                 'reduce_order': True, 'order_step': 1}
    metric = {'id': METRIC, 'name': 'Checkout value outside 0–20', 'source_id': source['id'],
              'query': 'anomaly_demo_value{service="checkout"}', 'enabled': True, 'delivery': True,
              'severity': 'warning', 'labels': {'team': 'platform'}, 'algorithm': algorithm, 'controls': controls}
    ensure('metrics', metric)
    template = {**metric, 'delivery': False}
    ensure('discoveries', {'id': '20000000-0000-4000-8000-000000000003', 'name': 'Checkout series',
                          'mode': 'query', 'source_id': source['id'], 'query': '{__name__="anomaly_demo_value"}',
                          'enabled': True, 'every_seconds': 30, 'stale_seconds': 120, 'limit': 10,
                          'cleanup_stale': True, 'template': template})
    for index, method in enumerate(('ar', 'shesd', 'intersection'), 4):
        settings = {**algorithm, 'method': method}
        if method == 'intersection':
            settings.update(quorum=2, members=[{**algorithm, 'method': 'ar'}, {**algorithm, 'method': 'shesd'}])
        ensure('metrics', {**metric, 'id': f'20000000-0000-4000-8000-{index:012d}',
                           'name': f'Preview {method.upper()} after warmup', 'enabled': False,
                           'query': 'last_over_time(anomaly_demo_value{service="checkout"}[1m])',
                           'delivery': False, 'algorithm': settings})
    print('Sandbox ready. Controls http://localhost:18081 | Aegis http://127.0.0.1:18080', flush=True)


if __name__ == '__main__':
    main()
