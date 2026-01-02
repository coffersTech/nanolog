import threading
import time
import json
import urllib.request
import urllib.error
import sys

class Worker(threading.Thread):
    def __init__(self, queue, server_url, api_key, instance_id):
        super().__init__()
        self.queue = queue
        self.server_url = server_url.rstrip('/')
        self.api_key = api_key
        self.instance_id = instance_id
        self.daemon = True
        self.stop_event = threading.Event()

    def run(self):
        batch = []
        last_send_time = time.time()

        while not self.stop_event.is_set() or not self.queue.empty():
            try:
                # Calculate time to wait
                now = time.time()
                time_since_last_send = now - last_send_time
                timeout = max(0, 1.0 - time_since_last_send)

                try:
                    record = self.queue.get(timeout=timeout)
                    batch.append(record)
                except Exception: # Empty queue after timeout
                    pass

                # Check if we should send
                if len(batch) >= 100 or (len(batch) > 0 and (time.time() - last_send_time) >= 1.0):
                    self._send_batch(batch)
                    batch = []
                    last_send_time = time.time()

            except Exception as e:
                # Fail safe to avoid crashing the thread main loop
                sys.stderr.write(f"NanoLog Worker Error: {e}\n")

        # Flush remaining
        if batch:
            self._send_batch(batch)

    def _send_batch(self, batch):
        try:
            data = json.dumps(batch).encode('utf-8')
            req = urllib.request.Request(
                f"{self.server_url}/api/ingest/batch",
                data=data,
                method='POST'
            )
            req.add_header('Content-Type', 'application/json')
            req.add_header('Authorization', f'Bearer {self.api_key}')
            req.add_header('X-Instance-ID', self.instance_id)

            with urllib.request.urlopen(req) as response:
                if response.status != 200:
                    sys.stderr.write(f"NanoLog Send Failed: HTTP {response.status}\n")
        except Exception as e:
            sys.stderr.write(f"NanoLog Network Error: {e}\n")

    def stop(self):
        self.stop_event.set()
