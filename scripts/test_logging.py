#!/usr/bin/env python3
"""Verify TASK-2001 logging against a built server and local PostgreSQL."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request

root = Path(__file__).resolve().parents[1]
binary = os.getenv('CATALOG_BINARY', '/tmp/catalog-task2001')
report = Path(os.getenv('LOG_REPORT_DIR', '/tmp/task2001-logs'))
report.mkdir(parents=True, exist_ok=True)
ansi = re.compile(r'\x1b\[[0-9;]*m')


def request(base, method, path):
    req = urllib.request.Request(base + path, method=method)
    try:
        response = urllib.request.urlopen(req, timeout=3)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        return response.status, response.read().decode()


for level, port in [('debug', 18083), ('info', 18084)]:
    env = dict(os.environ, APP_MONITOR_LOG_LEVEL=level,
               APP_PROCESSOR_WEB_SERVER_LISTEN_PORT=str(port))
    base = 'http://127.0.0.1:' + str(port)
    log_path = report / (level + '.txt')
    with log_path.open('w') as output:
        process = subprocess.Popen([binary], cwd=root, env=env, stdout=output, stderr=subprocess.STDOUT)
        try:
            for _ in range(100):
                if process.poll() is not None:
                    raise RuntimeError(log_path.read_text())
                try:
                    assert request(base, 'GET', '/health') == (200, 'ok')
                    break
                except (urllib.error.URLError, ConnectionError):
                    time.sleep(0.1)
            else:
                raise RuntimeError('server did not become ready')
            status, _ = request(base, 'POST', '/v1/category/list')
            assert status == 200
            status, body = request(base, 'DELETE', '/v1/category/invalid-guid')
            assert status == 400 and json.loads(body) == {'error': 'incorrect parameters'}
            assert request(base, 'GET', '/health') == (200, 'ok')
            time.sleep(0.1)
            logs = ansi.sub('', log_path.read_text())
            request_lines = [line for line in logs.splitlines() if ' finished' in line]
            assert not any('/health' in line for line in request_lines)
            successes = [line for line in request_lines if 'finished with no error' in line]
            failures = [line for line in request_lines if 'finished (or aborted) with error' in line]
            assert len(successes) == (1 if level == 'debug' else 0), logs
            assert len(failures) == 1 and 'ERR' in failures[0], logs
            assert all('exec_time=' in line and 'client_ip=' in line for line in request_lines)
            if successes:
                assert 'DBG' in successes[0]
            (report / (level + '-requests.txt')).write_text('\n'.join(request_lines) + '\n')
            print('PASS', level, 'success/error levels, client_ip, exec_time, health skipped')
            if level == 'debug':
                api_env = dict(env, CATALOG_URL=base, API_REPORT=str(report / 'api-results.json'))
                result = subprocess.run([sys.executable, 'scripts/test_api_binding.py'], cwd=root,
                                        env=api_env, capture_output=True, text=True, check=True)
                assert len(json.loads((report / 'api-results.json').read_text())) == 67
                print('PASS 67 API regression checks')
        finally:
            process.terminate()
            process.wait(timeout=10)
    log_path.write_text(ansi.sub('', log_path.read_text()))

result = subprocess.run([binary], cwd=root, env=dict(os.environ, APP_MONITOR_LOG_LEVEL='invalid'),
                        capture_output=True, text=True, timeout=10)
assert result.returncode == 1 and 'Invalid log level' in result.stdout + result.stderr
print('PASS invalid log level fails at startup')
print('Logs:', report)
