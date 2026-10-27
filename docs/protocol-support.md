# Protokol desteği

Her protokol kendi dinleyicisindedir. HTTP rota tablosu TCP, UDP veya TLS geçişiyle karışmaz.

| Protokol | Dinleyici | Ne yapar |
| --- | --- | --- |
| HTTP/1.1 | 10000 | Düz metin. WebSocket yükseltmesi de açıktır. |
| HTTPS | 8443 | Yerel CA ile TLS 1.2/1.3. ALPN `h2` ve `http/1.1`. |
| HTTP/2 | 10002 | Cleartext HTTP/2. |
| WebSocket | 10003 | HTTP/1 yükseltmesi. |
| SSE | 10005 | Zaman aşımı kapalı olay akışı. |
| gRPC | 10004 | HTTP/2 upstream. |
| TCP | 10020 | TCP proxy. HTTP yolu yok. |
| UDP | 10021 | UDP proxy. |
| TLS geçişi | 8444 | SNI eşleşir, TLS sonlandırılmaz. |

Yerel sertifika `make bootstrap` ile `.local/certs` altına üretilir. Sunucu adı `*.test.local` biçimindedir; tek noktalı `*.local` jokerini OpenSSL eşleştirmez. Bu public ACME veya kurumsal sertifika sağlayıcısı değildir. Anahtar imzalı pakete girmez; agent dosyadan SDS sırrı üretir.

Düğüm 1 bu portları `127.0.0.1` üzerinde yayınlar: HTTPS 18443, HTTP/2 10012, WebSocket 10013, gRPC 10014, SSE 10015, TCP 10020, UDP 10021, TLS geçişi 18444. `127.0.0.1:10000` HTTP edge olmaya devam eder.
