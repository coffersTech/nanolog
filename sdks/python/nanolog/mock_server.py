import http.server
import socketserver
import json
import sys

PORT = 8088

class MockRequestHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        print(f"\n[MockServer] Received POST request to {self.path}")
        print(f"[MockServer] Headers: {self.headers}")
        
        try:
            data = json.loads(post_data.decode('utf-8'))
            print(f"[MockServer] Body: {json.dumps(data, indent=2)}")
        except Exception as e:
            print(f"[MockServer] Body (Raw): {post_data}")
            print(f"[MockServer] Error decoding JSON: {e}")

        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'{"status":"ok"}')
        sys.stdout.flush()

    def log_message(self, format, *args):
        # Silence default logging
        return

if __name__ == "__main__":
    print(f"Mock server starting on port {PORT}...")
    with socketserver.TCPServer(("", PORT), MockRequestHandler) as httpd:
        print("Mock server running.")
        sys.stdout.flush()
        try:
            httpd.serve_forever()
        except KeyboardInterrupt:
            pass
        httpd.server_close()
