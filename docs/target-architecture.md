# Merkezi API Gateway — hedef mimari ve uygulama sözleşmesi

Tarih: 2026-10-27
Durum (2026-10-27): Merkezi /api/{serviceSlug}/ yönlendirmesi, otomatik path rewrite ve panel servis linkleri uygulanıp public HTTPS üzerinde doğrulandı. Servis bazlı auth, rate limit, observability ve çok-host HA gelecekteki geliştirmelerdir.

## Amaç

Onlarca veya yüzlerce API'yi tek bir merkezi gateway üzerinden erişilebilir, yönetilebilir ve izlenebilir kılmak. Her yeni backend için ayrı Caddy / Nginx dosyası ya da manuel reverse proxy değişikliği yapılmayacak. Backend'ler internetten doğrudan erişilebilir olmayacak.

## Tek giriş adresi

- Yönetim arayüzü (mevcut): https://api-gateway.mehmet.tech/ ve /login
- Ayrılmış veri yolu: https://api-gateway.mehmet.tech/api/{serviceSlug}/{upstreamPath}
- Mevcut servis 1: /api/catalog-api/api/info -> backend-a:8080/api/info
- Mevcut servis 2: /api/orders-api/api/info -> backend-b:8080/api/info
- Yeni bir API adı api3 seçildiğinde /api/api3/... yolu otomatik oluşur.
- Mevcut api1.mehmet.tech ve api2.mehmet.tech host yönlendirmeleri geriye dönük uyumluluk için korunur.
- /api/* public veri düzlemi; /gateway-api/* yönetim API'sidir. /api ve /gateway-api sınırları birbirine karışmayacak.
- İleride yönetim paneli için ayrı yönetim hostname'i, OIDC ve erişim kısıtı (VPN/Zero Trust) önerilir.

## Veri akışı

İstemci -> Caddy (TLS, tek public ingress) -> Envoy edge -> Envoy data-plane node 1/2 -> private backend.

Yönetici -> web panel -> authenticated control API -> PostgreSQL -> imzalı revision -> agent -> Envoy xDS.

**Kullanıcı trafiği** kontrol API'si, agent veya PostgreSQL üzerinden proxy edilmez. Kontrol düzlemi erişilemezken son doğrulanmış route'lar hizmet vermeye devam eder.

## Merkezi yönetim ve servis modeli

- Her servis bağımsız kayıt: benzersiz slug, hedef endpoint listesi, hostnames (isteğe bağlı), methods, path rules, environment/team/owner, TLS policy, route version.
- Slug benzersizliği aynı yayın alanında kontrol edilir; /api/<slug>/ ile benzer adlar (/api/api1 ile /api/api11 gibi) karışmaz.
- Path strip / rewrite varsayılanı: dış /api/{slug}/foo -> backend /foo; son slash, query, URL encoding, dot segments, encoded slash davranışı açıkça test edilir.
- Kayıt/düzenleme işlemi draft -> validate -> diff -> publish -> xDS ACK -> active durumuna ilerler; NACK'te son iyi konfigürasyon korunur.
- Bir servisin yayımı diğer servisleri veya başka listener'ları etkilemez. Çakışan route / slug, tekrarlı host, geçersiz hedef, reserved path publish öncesi reddedilir.
- Aynı public hostname altında farklı servislerin ayrı path'lere sahip olabilmesi için mevcut host-level exclusivity kontrolü yeniden tasarlanmalıdır.
- Envoy xDS aynı hostname için ilgili servis route'larını bir virtual host altında birleştirir. Caddy yalnız statik /api/* -> edge proxy işlemini yapar; **servis başına Caddy konfigürasyonu oluşmaz**.
- Temel policy: JWT/OIDC/API key doğrulaması (servis bazında), method/path ACL, CORS, request size, rate limit, IP allow/deny, upstream timeout, circuit breaker, health checks. Güvensiz/tekrarlanamaz istekler rastgele retry edilmez.
- Audit: kim, hangi servise, hangi route/policy'i ne zaman yayımladı; geri alma imkânı.
- Observability: RPS, HTTP status/4xx/5xx, p50/p95/p99, active requests, upstream health, route/cluster bazlı log ve izleme. Secret/key/token loglara yazılmaz.

## Uygulama sırası

1. API sözleşmesi ve DB migration: public serviceSlug, path rewrite/strip, conflict/validation; backward compatibility.
2. Compiler: aynı public domain üzerinde çok sayıda prefix, exact path ve method route; longest-prefix/explicit priority; Envoy virtual-host gruplama.
3. Tek Caddy /api/* kuralı -> Envoy edge (panel ve /gateway-api yönetim API'si bağımsız).
4. UI: servis eklerken public slug ve yayın URL'sini göster; aynı API ekranından route'ları, policy'leri ve upstream'leri düzenle.
5. api1 ve api2 için /api/api1/api/info ve /api/api2/api/info üzerinden HTTP 200, doğru instance, TLS doğrulaması ve iki düğüm failover testleri.
6. Yetkilendirme, dağıtık rate limit, metrik/trace, çoklu host/region HA ve bağımsız PostgreSQL yedekliliği.

## Başarı kriterleri

- Yönetim arayüzünden yeni bir servis ekleyip yayımlamak, ingress/Caddy dosyasında değişiklik gerektirmemeli.
- Yalnızca gateway public olmalı; backend portları public olmamalı.
- Bir servis silindiğinde /api/{slug}/... 404, diğerleri etkilenmeden 200.
- Yanlış host, reserved control-plane path, çakışan slug, hatalı backend güvenli şekilde reddedilmeli.
- Envoy node kesintisinde aktif rotalar devam etmeli; bunun tek host HA testi olduğu belirtilmeli.
- Yönetim API / DB erişilemezken yayımlanmış data-plane trafiği sürmeli.
- Yeni public yol çalışmadan eski api1/api2 domainleri kaldırılmamalı.

## Mevcut durum (2026-10-27)

- Public api1.mehmet.tech ve api2.mehmet.tech -> Caddy -> Envoy edge -> iki node -> backend: PASS.
- api-gateway.mehmet.tech: mevcut web kontrol paneli ve authenticated yönetim API.
- Tek gateway alan adında dinamik /api/{slug} yönlendirme: YOK; yukarıdaki uygulama sırası gereklidir.
- Çok sunuculu/çok lokasyonlu HA, auth gateway policy ve kapsamlı observability: henüz doğrulanmadı.
