frontend in
	bind *:9090
	bind *:8008
	bind *:8443
	use_backend metrics if { dst_port 9090 }
	use_backend webhook-https if { dst_port 8443 }
	use_backend webhook-http if { dst_port 8008 }

backend webhook-https
  server local-webhook ${IP_ADDR}:8443
  http-request deny deny_status 429 if { sc_http_req_rate(0) gt 10 }

backend metrics
  server metrics ${IP_ADDR}:9090

backend webhook-http
  server local-webhook ${IP_ADDR}:8008

defaults
  log global
  log 127.0.0.1 local0 debug
  option httplog
  timeout connect 5000
  timeout client  50000
  timeout server  50000

