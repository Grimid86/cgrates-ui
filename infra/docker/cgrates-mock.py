from http.server import HTTPServer, BaseHTTPRequestHandler
import json

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length)
        req = json.loads(body)
        method = req.get('method', '')
        params = req.get('params', {}) if isinstance(req.get('params'), dict) else {}

        if method == 'ApierV1.GetAccount':
            result = {
                "ID": params.get('Account', 'test'),
                "BalanceMap": {
                    "*monetary": [{"Value": 100.50, "ID": "main_monetary", "Type": "*monetary"}],
                    "*data": [{"Value": 2048, "ID": "main_data", "Type": "*data"}],
                    "*voice": [{"Value": 300, "ID": "main_voice", "Type": "*voice"}],
                    "*sms": [{"Value": 50, "ID": "main_sms", "Type": "*sms"}]
                }
            }
        elif method == 'SMGv1.GetActiveSessions':
            result = [
                {"MSISDN": "79161234567", "OriginHost": "127.0.0.1", "Destination": "79169876543", "Duration": 120, "StartTime": "2026-05-20T10:00:00Z"},
                {"MSISDN": "79169876543", "OriginHost": "127.0.0.1", "Destination": "79161234567", "Duration": 45, "StartTime": "2026-05-20T10:05:00Z"}
            ]
        elif method == 'ApierV1.GetCDRs':
            result = []
        elif method == 'ApierV1.AddBalance':
            result = "OK"
        else:
            result = {}

        resp = {"jsonrpc": "2.0", "result": result, "id": req.get("id", 1)}
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(resp).encode())

    def log_message(self, format, *args):
        pass

if __name__ == '__main__':
    print("CGRateS Mock starting on :2012")
    server = HTTPServer(('0.0.0.0', 2012), Handler)
    server.serve_forever()
