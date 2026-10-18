"""Local-only metric exporter, scenario controls and synthetic chat delivery sink."""
import json
import math
import threading
import time
import uuid
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.error import HTTPError
from urllib.parse import urlparse
from urllib.request import Request, urlopen

DATA = Path('/data/state.json')
LOCK = threading.RLock()
STATE = {'scenario': 'healthy', 'messages': []}
if DATA.exists():
    STATE.update(json.loads(DATA.read_text()))


def save():
    DATA.parent.mkdir(parents=True, exist_ok=True)
    temporary = DATA.with_suffix('.tmp')
    temporary.write_text(json.dumps(STATE))
    temporary.replace(DATA)


def values():
    with LOCK:
        first = {'healthy': 10, 'spike': 35, 'dip': -10}[STATE['scenario']]
    return {'api-1': round(first + .2 * math.sin(time.time() / 8), 3), 'api-2': 10}


def send_alert():
    event = str(uuid.uuid4())
    body = {'status': 'firing', 'labels': {'alertname': 'LocalWebhookTest', 'team': 'platform',
                                    'severity': 'warning', 'instance': event},
        'annotations': {'summary': 'Ordinary alert from the local sandbox'},
        'startsAt': datetime.now(timezone.utc).isoformat()}
    request = Request('http://aegis:3000/api/v1/alerts/webhook', json.dumps(body).encode(),
                      {'Content-Type': 'application/json', 'X-Aegis-Webhook-Secret': 'local-sandbox-webhook-only'})
    with urlopen(request, timeout=15) as response:
        return json.load(response)


class Handler(BaseHTTPRequestHandler):
    def reply(self, value, status=200, content_type='application/json'):
        raw = value if isinstance(value, bytes) else json.dumps(value).encode()
        self.send_response(status)
        self.send_header('Content-Type', content_type)
        self.send_header('Content-Length', str(len(raw)))
        self.send_header('Cache-Control', 'no-store')
        self.send_header('X-Content-Type-Options', 'nosniff')
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self):
        path = urlparse(self.path).path
        if path == '/':
            return self.reply(Path('/sandbox/index.html').read_bytes(), content_type='text/html; charset=utf-8')
        if path == '/metrics':
            lines = ['# HELP anomaly_demo_value Synthetic application value.', '# TYPE anomaly_demo_value gauge']
            lines += [f'anomaly_demo_value{{service="checkout",node="{node}",team="platform"}} {value}'
                      for node, value in values().items()]
            return self.reply(('\n'.join(lines) + '\n').encode(), content_type='text/plain; version=0.0.4; charset=utf-8')
        if path == '/api/state':
            with LOCK:
                return self.reply({**STATE, 'values': values(), 'tenant': 'local-sandbox'})
        if path == '/healthz':
            return self.reply({'healthy': True})
        if path.endswith('/token'):
            return self.reply({'status': 'ok', 'result': 'local-sandbox-token'})
        if path == '/api/v1/botx/chats/personal':
            return self.reply({'status': 'ok', 'result': {'group_chat_id': 'local-sandbox-chat'}})
        return self.reply({'error': 'not_found'}, 404)

    def do_POST(self):
        path = urlparse(self.path).path
        try:
            length = int(self.headers.get('Content-Length', '0'))
            if length < 0 or length > 1048576:
                return self.reply({'error': 'body_limit'}, 413)
            if path == '/api/auth.test':
                return self.reply({'ok': True, 'user_id': 'U_LOCAL_SANDBOX', 'team_id': 'T_LOCAL_SANDBOX'})
            # JSON-only controls plus Origin validation prevent cross-site form mutations.
            if self.headers.get('Content-Type', '').split(';')[0] != 'application/json':
                return self.reply({'error': 'json_required'}, 415)
            origin = self.headers.get('Origin')
            if origin and origin not in ('http://localhost:18081', 'http://127.0.0.1:18081'):
                return self.reply({'error': 'origin_forbidden'}, 403)
            body = json.loads(self.rfile.read(length) or '{}')
            if not isinstance(body, dict):
                return self.reply({'error': 'object_required'}, 400)
        except (ValueError, UnicodeDecodeError):
            return self.reply({'error': 'invalid_json'}, 400)
        if path == '/api/scenario':
            if body.get('scenario') not in ('healthy', 'spike', 'dip'):
                return self.reply({'error': 'invalid_scenario'}, 400)
            with LOCK:
                STATE['scenario'] = body['scenario']
                save()
            return self.reply({'scenario': body['scenario']})
        if path == '/api/alert':
            try:
                return self.reply(send_alert(), 202)
            except HTTPError as error:
                return self.reply({'error': 'aegis_rejected_alert', 'upstream_status': error.code}, 502)
            except (OSError, ValueError):
                return self.reply({'error': 'aegis_unavailable'}, 502)
        if path in ('/api/chat.postMessage', '/api/v4/botx/notifications/direct'):
            provider = 'slack' if path == '/api/chat.postMessage' else 'express'
            with LOCK:
                STATE['messages'].append({'provider': provider, 'at': datetime.now(timezone.utc).isoformat(), 'payload': body})
                STATE['messages'] = STATE['messages'][-100:]
                save()
            if provider == 'slack':
                return self.reply({'ok': True, 'channel': body.get('channel', 'local-sandbox'), 'ts': str(time.time())})
            return self.reply({'status': 'ok', 'result': {'sync_id': str(uuid.uuid4())}})
        return self.reply({'error': 'not_found'}, 404)

    def log_message(self, *args):
        pass


if __name__ == '__main__':
    ThreadingHTTPServer(('0.0.0.0', 8081), Handler).serve_forever()
