# Test sonuçları

Tarih: 2026-10-26T07:46:30Z
Ortam: Docker Compose, tek makine, iki gateway düğümü ve yerel edge. Bu sonuç rack, veri merkezi veya üretim HA kanıtı değildir.

| Test | Sonuç | Not |
| --- | --- | --- |
| Arayüz/API servisi iki backend'e gider | PASS | `e2e-1793000746.local` için `/a` backend-a, POST `/b` backend-b. Bilinmeyen host 404. İki düğüm de yanıtladı. |
| Viewer yayım yapamaz | PASS | HTTP 403. |
| Aynı idempotency anahtarı ikinci yayım açmaz | PASS | Revision 23 tekrarlandı. |
| Yayım farkı | PASS | Taslakta `/a` eklenecek olarak göründü. |
| Elle geri alma | PASS | Revision 23 paketinden revision 28 üretildi. İkinci hostname kalktı. Güncel pakete tekrar dönüş 409. Viewer 403. |
| Yönetim API durunca mevcut trafik sürer | PASS | API konteyneri durduruldu. |
| Merkez kapalıyken iki düğüm yeniden başlar | PASS | Doğrulanmış diskten iki düğüm ve edge trafik verdi. |
| Bozuk yerel önbellek ve merkez kapalı | PASS | İki düğümün dosyaları bozulunca HTTP 200 alınmadı. API dönünce trafik toparlandı. |
| Bir düğüm durunca yeni bağlantı | PASS | gw-local-1 durdurulunca edge 724 ms içinde gw-local-2 üzerinden yanıt verdi. Ortak üretim VIP'i yok. |
| İmza ve çakışma birim testleri | PASS | `go test ./...` |
| TLS, yanlış SAN, sertifika yenileme | NOT RUN | Public TLS yok. Sağlayıcı bilinmiyor. |
| NACK, canary, eşzamanlı yazıcı | PASS | Aynı expectedVersion ve iki idempotency anahtarı 200 ve 409 verdi. Canary revision 25 yalnız gw-local-1. NACK revision 26 ACK'i 25 bıraktı; disk current bu revision olmadı. Edge kanıtı değildir. |
| WebSocket, SSE, gRPC, büyük yükleme | PASS | 2026-10-23 düğüm 1 ölçümü. Bu koşu protokolleri yeniden ölçmedi. Büyük yükleme ayrıca NOT RUN. |
| CSRF, sahte istemci IP, SSRF uçtan uca | NOT RUN | Loopback ve metadata adresi birim testinde reddedilir. |
| OIDC, kota ve WAF | NOT RUN | Kimlik sağlayıcı ve bu bağımlılıklar yok. |
| Telemetri diski dolu | NOT RUN | Compose log sınırı 10m x 3 tanımlı, doluluk deneyi yapılmadı. |
| PostgreSQL primary ve quorum | NOT RUN | Tek PostgreSQL var. |
| Yük testi ve eklenen gecikme | NOT RUN | Ölçüm yok. |
| Yedekten geri dönüş | NOT RUN | Yedek alınmadı. |
| Lokasyon kaybı | NOT RUN | İkinci lokasyon yok. |

## Protokoller, 2026-10-23

Düğüm 1 üzerinde, yayım 16 ve sonrası. Host portları `127.0.0.1` üzerindedir. Bu tablo bu koşuda yeniden ölçülmedi.

| Protokol | Sonuç | Not |
| --- | --- | --- |
| HTTPS | PASS | `https-1792682249.test.local:18443`, yerel CA, HTTP/2, `instance=backend-a`. |
| HTTP/2 | PASS | Port 10012, prior knowledge. |
| WebSocket | PASS | Port 10013, 101 ve `instance=backend-a`. |
| SSE | PASS | Port 10015, `data: instance=backend-a`. |
| gRPC | PASS | Port 10014, health `serving`. |
| TCP | PASS | Port 10020, `/healthz` gövdesi `ok`. |
| UDP | PASS | `envoy:10021` docker ağında `pong`. Hosttaki `127.0.0.1:10021` bu Docker Desktop oturumunda yanıt vermedi. |
| TLS geçişi | PASS | Port 18444 sertifikası backend sertifikasıyla aynı. Gövde `passthrough instance=backend-a`. |



## Public HTTPS test laboratuvarı — 2026-10-27

Ortam: `ai.mehmet.tech`; mevcut Caddy (TLS), iki Envoy gateway düğümü, Envoy edge, PostgreSQL, yönetim API, Next.js paneli. **Tek fiziksel sunucu**; veri merkezi/lokasyon HA kanıtı değildir.

| Test | Sonuç | Açıklama |
| --- | --- | --- |
| Public panel `https://api-gateway.mehmet.tech/` | PASS | HTTP 200, geçerli HTTPS; `/login` ve `/test-lab` 200. |
| Public API 1 `https://api1.mehmet.tech/api/info` | PASS | HTTP 200, `instance=backend-a`, Catalog API. |
| Public API 2 `https://api2.mehmet.tech/api/info` | PASS | HTTP 200, `instance=backend-b`, Orders API. |
| Yönetim API'si üzerinden route yayımı | PASS | Yayın revizyonları 1 ve 2; iki gateway düğümü de güncel revizyonu ACK etti. |
| Her düğüme doğrudan istek | PASS | `127.0.0.1:10010` ve `:10011`; iki host da doğru backend'e yönlendi. |
| Yetkisiz lab API isteği | PASS | Public panel proxy'sinde 401. |
| Agent API public engellemesi | PASS | `/gateway-api/api/v1/agent/status` 404. |
| Caddy validasyonu | PASS | Mevcut Caddyfile yedeklendi, doğrulandı ve reload edildi. |
| Envoy edge konfigürasyon validasyonu | PASS | `envoy --mode validate` başarılı. |
| Node 1 kesinti testi | PASS | Node 1 kapalıyken 32/32 HTTPS yanıtı beklenen backend'den 200. |
| Node 2 kesinti testi | PASS | Node 2 kapalıyken 32/32 HTTPS yanıtı beklenen backend'den 200. |
| Node'ları geri başlatma | PASS | İki node ve iki public API tekrar sağlıklı. |

İlk denemede node 2 kapatılırken tekil geçici 503 gözlendi. Edge'e yalnız `connect-failure,refused-stream` koşullarında (2 kez) tekrar deneme eklendi; keyfi upstream 5xx ve istek-sonrası reset'lerde tekrar yoktur. Sonraki testte iki node için toplam 64/64 public istek başarılıydı. Bu örneklem **sıfır kesinti garantisi değildir**.

Doğrulanmayanlar: bütün hostun veya Caddy'nin kaybı, birden fazla lokasyon, PostgreSQL failover, kalıcı yük testi, tarayıcı ile operatör giriş senaryosu, bağımsız WAF/OIDC entegrasyonu.

## Merkezi gateway route ve panel linkleri — 2026-10-27

Ortam: ai.mehmet.tech, tek host, iki Envoy node, Caddy HTTPS ingress.

| Test | Sonuç | Kanıt |
| --- | --- | --- |
| Public merkezi API1 linki | PASS | https://api-gateway.mehmet.tech/api/catalog-api/api/info -> HTTP 200, backend-a, backend path /api/info |
| Public merkezi API2 linki | PASS | https://api-gateway.mehmet.tech/api/orders-api/api/info -> HTTP 200, backend-b, backend path /api/info |
| Hostname uyumluluğu | PASS | api1.mehmet.tech/api/info ve api2.mehmet.tech/api/info hâlâ HTTP 200 |
| Rota izolasyonu | PASS | /api/catalog-api2/api/info ve /api/unknown/ 404 |
| Yönetim / agent izolasyonu | PASS | public /gateway-api/api/v1/agent/package 404 |
| Panel ve yeni servis ekranları | PASS | /login, /services, /services/new 200; Next.js build ve TypeScript PASS |
| İki agent sürümü | PASS | İlk derlemede agent-2 eskiydi; image yeniden derlenip konfigürasyon dump ve iki node /api/info backend yolu doğrulandı |
| Merkezi linklerin yönetim API'sinden dönmesi | PASS | catalog-api ve orders-api GET list/detail gatewayBaseUrl alanı doğru |
| Ek DNS olmadan servis taslağı | PASS | Geçici taslak POST/validate geçti, yayına alınmadan silindi |
| Envoy Node 1 failover | PASS | Node 1 kapalıyken merkezi public API'ler 20/20 HTTP 200, doğru /api/info yolu |
| Envoy Node 2 failover | PASS | Node 2 kapalıyken merkezi public API'ler 20/20 HTTP 200, doğru /api/info yolu |
| Public TLS | PASS | Sertifika doğrulamasıyla HTTP 200, -k kullanılmadı |

Not: Bu testler yük/kapasite testi, çok sunuculu/lokasyonlu HA, OIDC, API anahtarı bazlı erişim politikaları veya WAF doğrulaması değildir. Public test API'leri bu aşamada backend kimlik doğrulaması olmadan erişilebilir. Yönetim API'si Bearer kimlik doğrulaması gerektirir.

Yeni servis yayını: panel /services/new -> servis kısa adı (gateway slug), opsiyonel özel alan adı, backend/port ve route -> validate -> publish -> /api/{slug}/{path}. Özel DNS olmadan kullanıldığında dahili ve public olmayan bir hostname saklanır; Caddy her servis için yeniden düzenlenmez.

## Backend authentication — 2026-10-27

UI: Publish Wizard step 2 (Hedef) now offers outbound none/basic/bearer/custom-header authentication and optional required inbound header. Inbound X-Gateway-Key authentication is separate and can be combined.

Scope: applies to ALL HTTP routes and hostnames of that service (not individually per method or per route). Header presence validation does not compare the inbound header's value.

Management endpoint (operator): POST /api/v1/services/{id}/upstream-auth with JSON fields mode, username, secret, headerName, requiredClientHeader. An empty secret retains the previous encrypted secret only if the mode is unchanged. After updates, the service must be published to both nodes.

Secret handling: AES-256-GCM encrypted at rest in upstream_auth_cipher after DB migration 00007. API list/details never return the secret/cipher. Revisions and xDS per-route context contain ciphertext, not plaintext; private node agent decrypts only when authenticating requests and injects credentials into upstream headers. The management control-plane is not on the runtime traffic path.

Key: .local/keys/credential.key (UID 65532, mode 0400), read-only mounted to API and both agents. Securely back up this key separately from the database; without it, existing encrypted credentials will not be decryptable. Do not commit or share. Future Vault/KMS integration recommended.

Live POST tests on catalog-api; each revision ACKed by both Envoy nodes:
Basic, Bearer and custom header: central URL and legacy api1.mehmet.tech both returned 401 when X-Client-Id was missing, 200 with the header. Unit tests also verify exact Basic/Bearer/custom upstream injected values and deny missing/invalid inbound API keys.

After validation, catalog-api was restored to outbound auth none, no required client header, and original GET/HEAD methods at revision 14. orders-api was unchanged.

Not yet supported: per-route credential policies, client header value matching, OIDC access token acquisition/refresh, upstream mTLS and automatic secret rotation. The test backend does not enforce credentials itself; injection of exact values was verified at the local authorization layer through unit tests.
## 2026-10-27 — Endpoint, CIDR ve trafik telemetrisi geliştirmesi

### Canlıda test edilenler
- `gw-speedtest-1` node kaydı: merkezi nodes DB'de mevcut, location = speedtest.mehmet.tech, heartbeat yok; **aktif node değildir**.
- Endpoint bazlı rate limit yönetim API'si: POST /api/v1/services/{id}/routes/{routeId}/rate-limit, {requestsPerSecond: 0..10000}; sıfır servis kotasını devralır.
- Servis IP allowlist/denylist: POST /api/v1/services/{id}/ip-policy, {allow:[CIDR],deny:[CIDR]}; normalize/unique/CIDR validation, audit. Denylist daha öncelikli; allowlist doluysa yalnız eşleşen IP.
- Envoy gerçek client IP: HCM use_remote_address=true, xff_num_trusted_hops=2; ext_authz doğrudan XFF yerine Envoy'un overwrite ettiği x-envoy-external-address değerini okur.
- Yalnızca catalog-api'ye geçici rate limit=2 uygulanarak r17'de node1: 1x200/11x429, node2: 2x200/10x429 ölçüldü.
- r18'de test allowlist 192.0.2.0/24: normal, forged X-Forwarded-For, forged X-Envoy-External-Address, forged XFF chain senaryolarının tamamı HTTP 403.
- r19 ile test CIDR'ler ve endpoint rate kısıtları temizlendi, catalog/legacy HTTP 200. orders-api değiştirilmedi.
- Authz telemetry checks+denied: per-node atomik sayım, heartbeat ile DB'ye kaydediliyor, düğüm ekranından izleniyor. Agent yeniden başlamasında sayaç sıfırlanır.
- Gerçek Envoy HTTP counterları (total/2xx/4xx/5xx) ekleme: admin üzerinden Unix socket /var/run/gateway/admin.sock, Docker volume yalnız Envoy + ilgili agent arasında; HTTP admin portu public değil. Kontrol edilip runtime yayına alınacaktır. Histogram, servis/endpoint bazlı latency, WAF kayıtları veya global RPS değildir.

### Henüz tamamlanmayan ürün kriterleri
- Speedtest agent çalıştırılıp xDS ACK alınmadı; node yalnız pending olarak kayıtlı. Resmi Envoy imajı ve Compose dosyaları hazır, onaylı registry ve secret-manager üzerinden güvenli imaj/kimlik dağıtımı gereklidir.
- Gerçek iki makine failover, public ingress HA, merkezi DB HA ve remote backend erişimi doğrulanmadı.
- Coraza + OWASP CRS WAF, dağıtık global rate limit, JWT/OIDC SSO, tam Türkçe/İngilizce çeviri ve servis/endpoint bazlı p95/p99 metrikleri tamamlanmadı.
- Uygulama hâlâ test laboratuvarı; güvenlik ürünü/üretim sertifikasyonu olarak sunulamaz.


### 2026-10-27 — HTTP telemetri kabul testi tamamlandı
- Envoy node1 ve node2 stat admin arayüzü host/public porttan çıkarılıp Docker xDS volume üzerindeki `/var/run/gateway/admin.sock` UNIX soketine taşındı. Protobuf bootstrap validate PASS, her iki node HTTP 200.
- Agent, Envoy'un `/stats?format=json&usedonly` endpoint'ini Unix soketinden okuyor; sayısal olmayan istatistikleri güvenle atlıyor. İlk sürüm metin alanları nedeniyle bütün metrikleri sıfırlıyordu; JSON RawMessage ile düzeltildi.
- Düğüm heartbeat üzerinden `telemetry_http_requests`, `telemetry_http_2xx`, `telemetry_http_4xx`, `telemetry_http_5xx` kaydediliyor.
- Canlı DB doğrulama: gw-local-1 1257 HTTP istek / 1257 2xx; gw-local-2 1599 HTTP istek / 1599 2xx; iki node ACK r19. Bu sayılar **agent/Envoy başından itibaren birikimli** ve sistemin synthetic health probe trafiğini de içerir.
- Düğüm UI'da HTTP/2xx/4xx/5xx ve auth kontrol/red sayacı ayrı gösteriliyor; bu, henüz endpoint RPS/p95/p99, tracing ya da tarihsel retention değildir.
- speedtest gw-speedtest-1: node kaydı görünür ama hiçbir agent heartbeat/ACK yok; dağıtım tamamlandı denemez.

Güncel durumda **WAF, JWT/OIDC, global rate limiter, tam TR/EN dil kapsamı, production multi-host HA henüz yapılmadı**. Ürün hazır gibi gösterilmemelidir.

### 2026-10-27 — Servis bazlı Güvenlik ve Trafik görünümü
- Yeni endpoint: `GET /api/v1/services/{id}/security-status` (mevcut operator/viewer RBAC ile). Belirtilen servisin kayıtlı güvenlik politikasını ve her node için en son ACK edilmiş revizyondan WAF, API Key, auth ve IP CIDR durumunu döndürür.
- **Güvenlik:** Signed revision içindeki `upstreamAuthCipher` ve `authKeyHash` HTML/JSON'a aktarılmaz. Yalnızca auth türü ve yapılandırma durumu döner; Go unit test bu durumu doğrular.
- Servis listesinde ayrı `Güvenlik politikası` sütunu: kayıtlı WAF modu, rate limit açık/kapalı, client API Key ve CIDR sayısı, yayın bekliyorsa uyarı. Listede kaydedilmiş politika gösterilir; kesin canlı politika için servis detayındaki node karşılaştırmasına bakılır.
- Servis detayında `Güvenlik ve Trafik` özeti: WAF, yerel kota, client access, upstream auth, CIDR sayısı.
- Her düğümün heartbeat ve ACK durumları: acknowledged / outdated / offline / not_deployed ve ACK edilen revizyona göre **uygulanan** politika. Speedtest node'u henüz ACK üretmediği için `not_deployed` görünür.
- Endpoint kota karşılaştırması: kayıtlı etkin değerler vs. node ACK revizyonundaki değerler; kayıtlı endpoint override sıfır ise servis genel kotasını miras alır.
- HTTP/WAF sayaçları **servis bazlı değildir**, toplam node sayacı olarak açıkça etiketlendi; yanlış biçimde servis istatistiği olarak gösterilmez.
- Servis detayındaki CIDR virgül/yeni satır ayırıcısı düzeltildi.
- Go testleri ve Next.js production build başarılı. Production'da yönetim paneli ve iki test servisi sağlıklı.
- Bu geliştirme servislerin güvenlik ayarlarını değiştirmez; yalnız kayıtlı/ACK edilmiş politikaları görünür kılar.
